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
		Durable: true,
		Timeout: 3 * time.Second,
		Serial:  true,
	}
)

// Aggregate represents the identity of some aggregate in the domain.
type Aggregate struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Event is the top-level type that wraps the event data.
type Event struct {
	// Stream is the stream this event was published on.
	Stream string `json:"stream"`

	// ID is the globally unique ID of the event.
	ID string `json:"id"`

	// Type is the event type.
	Type string `json:"type"`

	// Time when the event was published.
	Time time.Time `json:"time"`

	// Data is the event data.
	Data Data `json:"data"`

	// Schema is an identifier of the data schema.
	Schema string `json:"schema"`

	// Client is the ID of the client that produced this event.
	Client string `json:"client"`

	// Cause is the ID of the event that caused/resulted in this event
	// being produced.
	Cause string `json:"cause"`

	// Aggregate is the identify of an aggregate this event applies to.
	Aggregate *Aggregate `json:"aggregate,omitempty"`

	// Meta supports arbitrary key-value information associated with the event.
	Meta map[string]string `json:"meta,omitempty"`

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

// Handler is the event handler type for creating subscriptions.
type Handler func(ctx context.Context, evt *Event, conn Conn) error

// Conn is a connection interface to the underlying event streams backend.
type Conn interface {
	// Publish publishes an event to the specified stream. It returns the ID of the event.
	Publish(stream string, evt *Event) (string, error)

	// Subscribe creates a subscription to the stream and associates the handler.
	Subscribe(stream string, handle Handler, opts *SubscriptionOptions) (Subscription, error)

	// Close closes the connection.
	Close() error
}

type Subscription interface {
	// Unsubscribe closes the subscription and resets the offset.
	Unsubscribe() error

	// Close closes the subscription and retains the offset.
	Close() error
}

type SubscriptionOptions struct {
	// Unique name of the subscriber. This is used to keep track of the
	// the offset of messages for a stream. This defaults to the stream name.
	Name string

	// If true, a new subscription will be send the entire backlog of events
	// in the stream. This useful for
	Backfill bool

	// If true, the stream offset will be tracked for the subscriber. Upon
	// reconnect, the next message from the offset will be received.
	Durable bool

	// If true and the subscriber had a durable subscription, this will reset the
	// durable subscription. The effect is that all events from the specified
	// start position or time will be replayed.
	Reset bool

	// If true, a new event will be processed only if/when the previous event
	// was handled successfully and acknowledged. If events should be processed
	// in order, one at a time, then this should be set to true.
	Serial bool

	// The maximum time to wait before acknowledging an event was handled.
	// If the timeout is reached, the server will redeliver the event.
	Timeout time.Duration
}
