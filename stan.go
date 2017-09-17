package eda

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/chop-dbhi/eda/internal/pb"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	stanpb "github.com/nats-io/go-nats-streaming/pb"
	"github.com/nats-io/nuid"
)

// resetDurable resets a durable subscription by name.
func resetDurable(conn stan.Conn, stream, queueName, durableName string) error {
	// Connect with the durable name to unsubscribe.
	sub, err := conn.QueueSubscribe(stream, queueName, func(*stan.Msg) {}, stan.DurableName(durableName))

	if err != nil {
		return err
	}

	return sub.Unsubscribe()
}

type stanSubscription struct {
	channel  string
	consumer string
	durable  bool
	conn     *stanConn
	sub      stan.Subscription
}

func (s *stanSubscription) Close() error {
	return s.sub.Close()
}

func (s *stanSubscription) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

// stanConn is an implementation of Conn.
type stanConn struct {
	logger Logger

	client  string
	cluster string

	nats *nats.Conn
	stan stan.Conn
}

// Close the underlying connection to the stream backend.
func (c *stanConn) Close() error {
	if err := c.stan.Close(); err != nil {
		c.logger.Printf("[%s] connection close error: %s", c.client, err)
	}

	// No returned error.
	c.nats.Close()

	return nil
}

func (c *stanConn) Publish(stream string, evt *Event) (string, error) {
	var (
		err      error
		datab    []byte
		encoding string
	)

	if evt == nil {
		evt = &Event{}
	}

	if evt.Data == nil {
		encoding = "nil"
	} else {
		encoding = evt.Data.Type()
		datab, err = evt.Data.Encode()
	}

	if err != nil {
		return "", err
	}

	id := nuid.Next()

	b, err := proto.Marshal(&pb.Event{
		Id:       id,
		Type:     evt.Type,
		Cause:    evt.Cause,
		Client:   c.client,
		Data:     datab,
		Encoding: encoding,
		Meta:     evt.Meta,
	})
	if err != nil {
		return "", err
	}

	// Publish event.
	err = c.stan.Publish(stream, b)
	if err != nil {
		return id, err
	}

	return id, nil
}

func (c *stanConn) Subscribe(stream string, handle Handler, opts *SubscriptionOptions) (Subscription, error) {
	if opts == nil {
		opts = &SubscriptionOptions{}
	} else {
		opts = &(*opts)
	}

	if opts.Timeout == 0 {
		opts.Timeout = stan.DefaultAckWait
	}

	// TODO: Any long-term issue with this?
	consumerName := opts.Name
	if consumerName == "" {
		consumerName = c.client
	}
	durableName := consumerName

	if opts.Reset {
		if err := resetDurable(c.stan, stream, consumerName, durableName); err != nil {
			return nil, err
		}
	}

	// Handler for the raw message.
	msgHandler := func(msg *stan.Msg) {
		var e pb.Event

		// Message sent on stream that is not a protobuf format.
		err := proto.Unmarshal(msg.Data, &e)
		if err != nil {
			c.logger.Printf("[%s] proto unmarshal failed: %s", c.client, err)
			return
		}

		dec := decodable{
			b:   e.Data,
			t:   e.Encoding,
			e:   true,
			enc: encMap[e.Encoding],
		}

		evt := &Event{
			Stream: msg.Subject,
			ID:     e.Id,
			Time:   time.Unix(0, msg.Timestamp),
			Type:   e.Type,
			Cause:  e.Cause,
			Client: e.Client,
			Data:   &dec,
			Meta:   e.Meta,
			msg:    msg,
		}

		// Use ack timeout as max context timeout to signal handler components.
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()

		// Recover from panic to properly close connection.
		defer func() {
			if err := recover(); err != nil {
				c.Close()
				panic(err)
			}
		}()

		// Handler error implies a timeout or implementation issue.
		if err := handle(ctx, evt); err != nil {
			c.logger.Printf("[%s] handler error: %s", c.client, err)
			return
		}

		// Couldn't acknowledge the event has been handled.
		// Bad subscription or bad connection.
		if err := msg.Ack(); err != nil {
			c.logger.Printf("[%s] ack failed: %s", c.client, err)
		}
	}

	// Map start position.
	var startPos stanpb.StartPosition

	if opts.Backfill {
		startPos = stanpb.StartPosition_First
	} else {
		startPos = stanpb.StartPosition_NewOnly
	}

	subOpts := []stan.SubscriptionOption{
		// Set the initial start position.
		stan.StartAt(startPos),

		// Use manual acks to manage errors.
		stan.SetManualAckMode(),
	}

	// Length of time to wait before the server resends the message.
	if opts.Timeout > 0 {
		subOpts = append(subOpts, stan.AckWait(opts.Timeout))
	}

	// Force messages to be processed in ordered with manual acking.
	if opts.Serial {
		subOpts = append(subOpts, stan.MaxInflight(1))
	}

	if opts.Durable {
		subOpts = append(subOpts, stan.DurableName(durableName))
	}

	qsub, err := c.stan.QueueSubscribe(
		stream,
		consumerName,
		msgHandler,
		subOpts...,
	)
	if err != nil {
		return nil, err
	}

	sub := &stanSubscription{
		channel:  stream,
		consumer: consumerName,
		conn:     c,
		sub:      qsub,
		durable:  opts.Durable,
	}

	return sub, nil
}

// Logger is a minimal interface required for internal logging.
// This is compatible with the stdlib log.Logger type.
type Logger interface {
	Print(v ...interface{})
	Printf(f string, v ...interface{})
}

type ConnectOptions struct {
	Logger Logger
}

func (o *ConnectOptions) Apply(opts ...ConnectOption) {
	for _, f := range opts {
		f(o)
	}
}

type ConnectOption func(o *ConnectOptions)

func WithLogger(l Logger) ConnectOption {
	return func(o *ConnectOptions) {
		o.Logger = l
	}
}

// Connect establishes a connection to the streaming backend.
func Connect(addr, cluster, client string, opts ...ConnectOption) (Conn, error) {
	o := &ConnectOptions{
		Logger: log.New(os.Stderr, "[eda] ", log.LstdFlags),
	}

	o.Apply(opts...)

	// Logging disabled. Re-initialize to discard.
	if o.Logger == nil {
		o.Logger = log.New(ioutil.Discard, "", 0)
	}

	nc, err := nats.Connect(
		addr,
		// Try reconnecting indefinitely.
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, err
	}

	// Initialize streaming connection.
	snc, err := stan.Connect(cluster, client, stan.NatsConn(nc))
	if err != nil {
		return nil, err
	}

	conn := stanConn{
		client:  client,
		cluster: cluster,
		logger:  o.Logger,
		stan:    snc,
		nats:    nc,
	}

	return &conn, nil
}
