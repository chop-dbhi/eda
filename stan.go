package eda

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
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

// stanConn is an implementation of Conn.
type stanConn struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *log.Logger
	errch  chan error

	subs []stan.Subscription

	client  string
	cluster string

	nats *nats.Conn
	stan stan.Conn
}

func (c *stanConn) handleInterrupt() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Kill, os.Interrupt)

	// Either an OS signal will be caught or
	// a programmatic cancel will be triggered.
	select {
	case s := <-sig:
		c.logger.Printf("signal received: %s", s)
		c.cancel()

	case <-c.ctx.Done():
		c.logger.Print("connection done")
	}

	c.errch <- nil
}

// monitor monitors the client connection.
func (c *stanConn) monitor() {
	ticker := time.Tick(5 * time.Second)

	connected := true

	for {
		select {
		case <-ticker:
			status := c.nats.Status()

			if status == nats.CONNECTED {
				if !connected {
					c.logger.Print("reconnected!")
					connected = true
				}
				break
			}

			connected = false

			switch status {
			case nats.DISCONNECTED:
				c.logger.Print("disconnected")
			case nats.CLOSED:
				c.logger.Print("closed")
			case nats.RECONNECTING:
				c.logger.Print("reconnecting")
			case nats.CONNECTING:
				c.logger.Print("connecting")
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// Close closes all subscriptions and the underlying connections.
func (c *stanConn) Close() error {
	// Shut down background routines.
	c.cancel()

	for _, s := range c.subs {
		if err := s.Close(); err != nil {
			c.logger.Printf("subscription close error: %s", err)
		}
	}

	if err := c.stan.Close(); err != nil {
		c.logger.Printf("connection close error: %s", err)
	}

	// No returned error.
	c.nats.Close()

	return nil
}

func (c *stanConn) Publish(stream string, typ string, data Encodable, opts ...PublishOption) (string, error) {
	var (
		err      error
		datab    []byte
		encoding string
	)

	if data != nil {
		// Check for decoable and use that directly.
		if dec, ok := data.(*decodable); ok {
			encoding = data.Type()
			datab, err = data.Encode()
		} else {
			encoding = data.Type()
			datab, err = data.Encode()
		}
	} else {
		encoding = "nil"
	}

	if err != nil {
		return "", err
	}

	var o PublishOptions
	o.Apply(opts...)

	id := nuid.Next()

	b, err := proto.Marshal(&pb.Event{
		Id:       id,
		Type:     typ,
		Cause:    o.Cause,
		Client:   c.client,
		Data:     datab,
		Encoding: encoding,
	})

	if err != nil {
		return "", err
	}

	// Publish event.
	err = c.stan.Publish(stream, b)
	if err != nil {
		return id, err
	}

	c.logger.Printf("publish(type=%v stream=%v)", typ, stream)

	return id, nil
}

func (c *stanConn) Subscribe(stream string, handle Handler, opts ...SubscriptionOption) (Subscription, error) {
	// Copy defaults.
	subOpts := *DefaultSubscriptionOptions
	subOpts.Apply(opts...)

	// TODO: Any long-term issue with this?
	consumerName := subOpts.ConsumerName
	if consumerName == "" {
		consumerName = c.client
	}
	durableName := consumerName

	if subOpts.Reset {
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
			c.errch <- err
			c.cancel()
			return
		}

		dec := decodable{
			b: e.Data,
			t: e.Encoding,
		}

		evt := &Event{
			ID:     e.Id,
			Time:   msg.Timestamp,
			Type:   e.Type,
			Cause:  e.Cause,
			Client: e.Client,
			Data:   &dec,
			msg:    msg,
		}

		// Use ack timeout as max context timeout to signal handler components.
		ctx, _ := context.WithTimeout(c.ctx, subOpts.AckTimeout)

		// Recover from panic to properly close connection.
		defer func() {
			if err := recover(); err != nil {
				c.cancel()
				c.Close()
				panic(err)
			}
		}()

		// Handler error implies a timeout or implementation issue.
		if err := handle(ctx, evt, c); err != nil {
			c.errch <- err
			c.cancel()
			return
		}

		// Couldn't acknowledge the event has been handled.
		// Bad subscription or bad connection.
		if err := msg.Ack(); err != nil {
			c.errch <- err
			c.cancel()
		}
	}

	// Map start position.
	var startPos stanpb.StartPosition

	switch subOpts.StartPosition {
	case StartPositionFirst:
		startPos = stanpb.StartPosition_First
	case StartPositionNew:
		startPos = stanpb.StartPosition_NewOnly
	default:
		return nil, fmt.Errorf("invalid start position: %d", subOpts.StartPosition)
	}

	subOptions := []stan.SubscriptionOption{
		// Set the initial start position.
		stan.StartAt(startPos),

		// Length of time to wait before the server resends the message.
		stan.AckWait(subOpts.AckTimeout),

		// Use manual acks to manage errors.
		stan.SetManualAckMode(),
	}

	if subOpts.Serial {
		subOptions = append(subOptions, stan.MaxInflight(1))
	}

	if subOpts.Durable {
		subOptions = append(subOptions, stan.DurableName(durableName))
	}

	sub, err := c.stan.QueueSubscribe(
		stream,
		consumerName,
		msgHandler,
		subOptions...,
	)
	if err != nil {
		return nil, err
	}

	c.logger.Printf("subscribe(id=%v stream=%v durable=%v serial=%v", consumerName, stream, subOpts.Durable, subOpts.Serial)

	c.subs = append(c.subs, sub)

	return sub, nil
}

func (c *stanConn) Wait() error {
	// Block until signaled done. Either due to an OS signal or error.
	<-c.ctx.Done()

	// Return nil or the error.
	return <-c.errch
}

// Connect establishes a connection to the event stream backend.
func Connect(ctx context.Context, addr, cluster, client string) (Conn, error) {
	ctx, cancel := context.WithCancel(ctx)

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

	var logger *log.Logger
	if DefaultLogger == nil {
		logger = log.New(ioutil.Discard, "", 0)
	} else {
		logger = &(*DefaultLogger)
	}

	conn := stanConn{
		ctx:     ctx,
		cancel:  cancel,
		client:  client,
		cluster: cluster,
		logger:  logger,
		stan:    snc,
		nats:    nc,
		errch:   make(chan error, 2),
	}

	// Start monitoring.
	go conn.monitor()

	// Handle OS signals.
	go conn.handleInterrupt()

	logger.Printf("connect(id=%v addr=%v cluster=%v)", client, addr, cluster)

	return &conn, nil
}

/*
type MultiSubscription struct {
	conn Conn
	subs   []Subscription
}

func (m *MultiSubscription) Close() error {
	for _, s := range subs {
		s.Close()
	}
}

func NewMultiSubscription(conn Conn, streams []string, handler stan.MsgHandler, opts ...SubscriptionOption) (*MultiSubscription, error) {
	subs := make([]Subscription, len(streams))

	// Create subscription per stream and fan-in events.
	for i, stream := range streams {
		sub, err := conn.Subscribe(stream, handler, opts...)
		if err != nil {
			return nil, err
		}

		subs[i] = sub
	}

	return &MultiSubscription{
		conn: Conn,
		subs:   subs,
	}, nil
}
*/
