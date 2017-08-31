package eda

import (
	"context"
	"log"
	"os"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
)

var (
	// Default logger used for connections.
	DefaultLogger = log.New(os.Stderr, "[eda] ", log.LstdFlags)

	// Default subscription options.
	DefaultSubscriptionOptions = &SubscriptionOptions{
		Durable:       true,
		AckTimeout:    3 * time.Second,
		Serial:        true,
		StartPosition: StartPositionFirst,
	}
)

// Event is the top-level type that wraps the event data.
type Event struct {
	// ID is the globally unique ID of the event.
	ID string `json:"id"`

	// Type is the event type.
	Type string `json:"type"`

	// Time is the time in nanoseconds with the event was published.
	Time int64 `json:"time"`

	// Data is the event data.
	Data Decodable `json:"data"`

	// Client is the ID of the client that produced this event.
	Client string `json:"client"`

	// Cause is the ID of the event that caused/resulted in this event
	// being produced.
	Cause string `json:"cause"`

	msg *stan.Msg
}

// IsType returns true if the event is one of the passed types.
func (e *Event) Is(types ...string) bool {
	for _, t := range types {
		if e.Type == t {
			return true
		}
	}

	return false
}

// PublishOption sets a publish option.
type PublishOption func(*PublishOptions)

// PublishOptions contains options when publishing an event.
type PublishOptions struct {
	// Cause is the ID of the causal event.
	Cause string
}

// Apply applies all option funcs to the publish options.
func (o *PublishOptions) Apply(opts ...PublishOption) {
	for _, f := range opts {
		f(o)
	}
}

// Cause sets the cause of the event being published.
func Cause(cause string) PublishOption {
	return func(o *PublishOptions) {
		o.Cause = cause
	}
}

// Handler is the event handler type for creating subscriptions.
type Handler func(ctx context.Context, evt *Event, conn Conn) error

// Conn is a connection interface to the underlying event streams backend.
type Conn interface {
	// Publish publishes an event to the specified stream.
	Publish(stream string, typ string, data Encodable, opts ...PublishOption) (string, error)

	// Subscribe subscribes to a stream.
	Subscribe(stream string, handle Handler, opts ...SubscriptionOption) (Subscription, error)

	// MultiSubscribe subscribes to multiple streams and applies the same handler.
	// MultiSubscribe(streams []string, handle Handler, opts ...SubscriptionOption) (Subscription, error)

	// Close closes the connection.
	Close() error

	// Wait
	Wait() error
}

type Subscription interface {
	// Unsubscribe closes the subscription and resets the offset.
	Unsubscribe() error

	// Close closes the subscription and retains the offset.
	Close() error
}

type StartPosition uint8

const (
	// Start with only new events in the stream.
	StartPositionNew = StartPosition(iota)

	// Start with the first event in the stream.
	StartPositionFirst
)

type SubscriptionOptions struct {
	// Unique name of the subscriber. This is used to keep track of the
	// the offset of messages for a stream. This defaults to the stream name.
	ConsumerName string

	// If true, the stream offset will be tracked for the consumer. Upon
	// reconnect, the next message from the offset will be received.
	Durable bool

	// If true and the consumer had a durable subscription, this will reset the
	// durable subscription. The effect is that all events from the specified
	// start position or time will be replayed.
	Reset bool

	// If true, a new event will be processed only if/when the previous event
	// was handled successfully and acknowledged.
	Serial bool

	// The maximum time to wait before acknowledging the consumer handled
	// the message. If the timeout is reached.
	AckTimeout time.Duration

	// Position specifies position in the stream to start from. This only applies
	// for non-durable streams, the first time a durable stream is created, or
	// if the durable stream is "reset".
	StartPosition StartPosition
}

func (o *SubscriptionOptions) Apply(opts ...SubscriptionOption) {
	for _, f := range opts {
		f(o)
	}
}

type SubscriptionOption func(*SubscriptionOptions)

func ConsumerName(s string) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.ConsumerName = s
	}
}

func Durable(b bool) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.Durable = b
	}
}

func Reset(b bool) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.Reset = b
	}
}

func Serial(b bool) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.Serial = b
	}
}

func AckTimeout(t time.Duration) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.AckTimeout = t
	}
}

func StartNew() SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.StartPosition = StartPositionNew
	}
}

func StartFirst() SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.StartPosition = StartPositionFirst
	}
}
