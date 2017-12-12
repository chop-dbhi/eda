# eda

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/chop-dbhi/eda)](https://goreportcard.com/report/github.com/chop-dbhi/eda) [![Build Status](https://travis-ci.org/chop-dbhi/eda.svg?branch=master)](http://travis-ci.org/chop-dbhi/eda) [![GoDoc](https://godoc.org/github.com/chop-dbhi/eda?status.svg)](http://godoc.org/github.com/chop-dbhi/eda)

eda is a library for implementing event-driven architectures. It provides a set of concrete types for message-driven architectures specifically as applied in CQRS and Event Sourcing. There is a core `Message` envelope used for wrapping events, commands, and queries. There is a `Reply` type that can be used in conjunction with commands and includes a status code and message in addition to data passed back from the handler. Both `Message` and `Reply` rely on the `Data` type which provides an easy way to wrap domain-specific message data and embed them within the message. All types have built-in support for marshaling and unmarshaling so they can be transmitted over the wire using a transport of your choice.

## Status

**The library is in an Alpha stage and looking feedback and discussions on API design and scope. Check out the [issues](https://github.com/chop-dbhi/eda/issues) for topics.**

## Install

Requires Go 1.9+

```
go get -u github.com/chop-dbhi/eda
```

## Usage Example

Below is a type that represents a domain event in a study recruitment application.

```go
// ScreeningAppointmentScheduled denotes a new appointment has been created
// for screening a subject for recruitment.
type ScreeningAppointmentScheduled struct {
  // ID of the subject
  SubjectId string
  // Phone number of the subject.
  SubjectPhone string
  // ID of the study recruiter who will conduct the screening.
  RecruiterId string
  // Scheduled time for the appointment.
  Time time.Time
  // Location of the appointment.
  Location string
}
```

Note that this event only contains information relevant to the domain and not boilerplate event metadata such as the time, unique ID, etc. This metadata is defined on the `eda.Message` envelope and enables encoding the domain event as data in the message.

```go
m := eda.Message{
  Type: "ScreeningAppointmentScheduled",
  Data: eda.JSON(&ScreeningAppointmentScheduled{
    SubjectId: "s-3932832",
    SubjectPhone: "555-102-1039",
    RecruiterId: "r-9430439",
    Time: time.Date(2018, 1, 23, 14, 0, 0, 0, time.Local),
    Location: "Research Building",
  }),
}
```

The use of `eda.JSON` will convert the domain event into a `eda.Data` type with a JSON encoding. Additional built-in codecs including `eda.Proto`, `eda.String`, and `eda.Bytes`. [Refer to the docs](https://godoc.org/github.com/chop-dbhi/eda/#Message) for additional fields that can be set on the `Message` type.

When the message is marshalled, the domain event will be internally marshalled to JSON. A call to `Marshal` will automatically generate a unique ID and set the current time on the message. These values can be set ahead of time if more control is needed.

```go
b, err := m.Marshal()
```

Internally the envelope is marshalled using a Protocol Buffer message type. This is for efficiency, but also allows for cross-language support.

At this point the bytes can be sent over the wire using a *messaging transport* such as NATS, NATS Streaming, Kafka, RabbitMQ, etc. To manually unmarshal, simply call unmarshal on an initialized `eda.Message` value and pass the raw bytes.

```go
var m eda.Message
err := m.Unmarshal(b)
```

To access the original domain event simply initialize a value of the corresponding type and decode it from the `Data` field.

```go
var e ScreeningAppointmentScheduled
err := m.Data.Decode(&e)
```

This will decode the original domain event into the initialized value. In practice, a common pattern is to switch on the `eda.Message.Type` field to select which concrete domain type to decode into.

```go
switch m.Type {
case "ScreeningAppointmentScheduled":
  var e ScreeningAppointmentScheduled
  err := m.Data.Decode(&e)

// handle other types..
}
```

For more information refer to the [docs](https://godoc.org/github.com/chop-dbhi/eda).

## Use Cases with Examples

All examples below use either NATS or NATS Streaming as the message transport. For use cases that support both transports, the choice of using one vs. the other comes down to whether messages can be dropped and/or messages are time-sensitive where delayed consumption would not be useful.

NATS provides a fast in-memory broadcast of messages over topics. If there are no consumers for that topic at the time of publishing, the message will be dropped. In this respect, the delivery guarantee is *at-most-once*.

NATS Streaming provides durability of messages with a configurable retention policy. This means that even if consumers are offline, they can connect and *catch-up* by consuming past messages. In contrast with NATS, the delivery guarantee is *at-least-once*.

Below are the minimal code required to establish a [NATS](https://godoc.org/github.com/nats-io/go-nats#Connect) and [NATS Streaming connection](https://godoc.org/github.com/nats-io/go-nats-streaming#Connect) connection, respectively. The connections (by variable name) will be referenced in the below code examples.

```go
// NATS.
nc, err := nats.Connect("nats://localhost:4222")
if err != nil {
  log.Fatal(err)
}
defer nc.Close()

// NATS Streaming.
sc, err := stan.Connect(
  // Name of the "cluster" to connect to. This is a parameter of the
  // NATS Streaming Server itself.
  "test-cluster",
  // Unique ID of this connection, referred to as the client-id.
  "test-client",
  stan.NatsURL("nats://localhost:4222"),
)
if err != nil {
  log.Fatal(err)
}
defer sc.Close()
```

### Async Message Broadcasting

This use case involves simply broadcasting information to subscribers. To publish a message:

```go
// Prepare the message.
m := &eda.Message{
  Type: "foo",
  Data: eda.String("some message"),
}

// Marshal the message. An error could only occur if
// the Data or Meta fields cannot be marshaled properly.
b, err := m.Marshal()
if err != nil {
  log.Fatal(err)
}

// Publish using NATS.
err := nc.Publish("messages", b)
if err != nil {
  log.Fatal(err)
}

// or...

// Publish using NATS Streaming.
err := sc.Publish("messages", b)
if err != nil {
  log.Fatal(err)
}
```

For completeness, below are exmaples of a consumer for both NATS and NATS Streaming. The boilerplate is the same. The primary different are the subscription options available.

#### NATS Consumer

```go
func run(nc *nats.Conn) error {
  done := make(chan error)

  sub, err := nc.Subscribe("messages", func(msg *nats.Msg) {
    var m eda.Message

    // Only occurs if the NATS message data is not eda.Message bytes..
    if err := m.Unmarshal(msg.Data); err != nil {
      done <- err
      return
    }

    // Decode the message data.
    var s string
    if err := m.Data.Decode(&s); err != nil {
      done <- err
      return
    }

    // Print the message.
    fmt.Println(s)
  })
  if err != nil {
    return err
  }
  defer sub.Close()

  // Handle signal.
  sig := make(chan os.Signal)
  signal.Notify(sig, os.Interrupt, os.Kill)

  // Wait for signal or done channel.
  select {
  case <-sig:
  case err <- done:
    return err
  }
  return nil
}
```

#### NATS Streaming Consumer

```go
func run(sc stan.Conn) error {
  done := make(chan error)

  // Subscription options.
  opts := []stan.SubscriptionOption{
    stan.SetManualAckMode(),
    // For "resumable" subscriptions, uncomment this option and specify
    // a unique name relative to the connection client-id.
    // stan.DurableName("test-consumer"),
  }

  sub, err := sc.Subscribe("messages", func(msg *stan.Msg) {
    var m eda.Message

    // Only occurs if the NATS message data is not eda.Message bytes..
    if err := m.Unmarshal(msg.Data); err != nil {
      done <- err
      return
    }

    // Do something with just the message..

    // Decode the message data.
    var s string
    if err := m.Data.Decode(&s); err != nil {
      done <- err
      return
    }

    // Print the message.
    fmt.Println(s)

    // Acknowledge receipt and consumption of the message.
    // This will proceed the offset for this particular consumer.
    if err := msg.Ack(); err != nil {
      done <- err
    }
  }, opts...)
  if err != nil {
    return err
  }
  defer sub.Close()

  // Handle signal.
  sig := make(chan os.Signal)
  signal.Notify(sig, os.Interrupt, os.Kill)

  // Wait for signal or done channel.
  select {
  case <-sig:
  case err <- done:
    return err
  }

  return nil
}
```

### Command Processing

A command expresses an intent to perform some action which may result in a state change. It comes in the form of a request that is sent to a *command handler* that can understand and execute the command on the client's behalf. Those familiar with HTTP, the verbs POST, PUT, DELETE, and PATCH are all commands (GET and HEAD are more synonymous with *queries* since they do not have side effects).

Commands should only be processed by a single handler rather than being broadcasted out to multiple independent consumers. Likewise, the success or failure of a sent command should be communicated back to the sender.

```go
m := &eda.Message{
  Type: "ScheduleAppointment",
  Data: eda.JSON(&Appointment{
    SubjectId: "s-3932832",
    SubjectPhone: "555-102-1039",
    RecruiterId: "r-9430439",
    Time: time.Date(2018, 1, 23, 14, 0, 0, 0, time.Local),
    Location: "Research Building",
  }),
}
```

In addition to the `Message` envelope to wrap the command, the a `Reply` type is provided that can be used by the command handler reply a status and optional data.

```go
const FailedPrecondition = 9

r := &eda.Reply{
  Status: FailedPrecondition,
  Message: "Appointment already scheduled in that time slot.",
}
```

The command above encodes a description of an appointment the client wants to be scheduled. NATS has native support for the request/reply pattern. The client code to send the request is as follows:

```go
// Require the processing take no longer than a second.
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

// Marshal and send command to the "scheduling" topic.
b, _ := c.Marshal()

// Returned is a NATS message or an error.
msg, _ := nc.RequestContext(ctx, "scheduling", b)

var r eda.Reply
r.Unmarshal(msg.Data)

// Do something with the reply which contains the status code and message
// and optional data.
```

The only difference between a call to `Publish` and a call to `Request`/`RequestContext` is that the message includes a unique topic the handler can send the reply back to. Thus the command handler is just another subscription that publishes the reply back to the topic:

```go
// Subscribe to the scheduling topic. Boilerplate elided.
nc.Subscribe("scheduling", func(msg *nats.Msg) {
  var m eda.Message
  m.Unmarshal(msg.Data)

  switch m.Type {
  case "ScheduleAppointment":
    var appt Appointment
    m.Data.Decode(*appt)

    // Delegate to specific function to handle the command.
    // The return value is a eda.Reply type that is published
    // back to the client.
    reply := scheduleAppointment(&appt)

    b, _ := r.Marshal()

    // Publish back to the client via the unique topic included with
    // the NATS message.
    nc.Publish(msg.Reply, b)
  }
})
```

### Event Sourcing

A common pattern which extends the above example is to have the command handler emit a set of events expressing the change in state (a new appointment being scheduled) when a command is sucessfully handled.

To maintain the decoupling of the delegated command handlers and the NATS layer, the `scheduleAppointment` function could return a set of events (wrapped in `eda.Message`) that are then published to a topic in order to communicate these changes.

```go
// Delegate to specific function to handle the command.
// The return value is a eda.Reply type that is published
// back to the client.
events, reply := scheduleAppointment(&appt)

// Publish events to a NATS Streaming topic (for persistence) which is
// independent of the NATS topics. Only NATS Streaming subscribers can
// consume this topic.
for _, e := range events {
  b, _ := e.Marshal()
  sc.Publish("scheduling", b)
}

b, _ := reply.Marshal()

// Publish back to the client via the unique topic included with
// the NATS message.
nc.Publish(msg.Reply, b)
```

## Resources

- Read about [use cases for persistent logs](https://dev.to/byronruth/use-cases-for-persistent-logs-with-nats-streaming)

## License

MIT
