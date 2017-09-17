# eda

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/chop-dbhi/eda)](https://goreportcard.com/report/github.com/chop-dbhi/eda) [![Build Status](https://travis-ci.org/chop-dbhi/eda.svg?branch=master)](http://travis-ci.org/chop-dbhi/eda) [![GoDoc](https://godoc.org/github.com/chop-dbhi/eda?status.svg)](http://godoc.org/github.com/chop-dbhi/eda)

eda is a library for implementing event-driven architectures. It provides a thin layer on top of backends that support ordered, durable streams with a publish/subscribe interface. The current implementation uses [NATS Streaming](https://github.com/nats-io/nats-streaming-server) as a backend.

## Status

**The library is in an Alpha stage and looking feedback and discussions on API design and scope. Check out the [issues](https://github.com/chop-dbhi/eda/issues) for topics.**

## Use Case

The primary use case this library is being designed to support are applications involving "domain events". That is, these events carry information about something that occurred in a domain model that must be made available for other consumers.

One application of this is as a building block for systems using CQRS pattern where events produced on the write side (a result of handling a command) need to get published so the read side can consume and update their internal indexes.

Another related use case is Event Sourcing which are generally spoken of in the context of an "aggregate". The pattern requires each aggregate instance to maintain it's own stream of events acting as an internal changelog. This stream is generally "private" from other consumers and requires having a single handler to apply events in order to maintain a consistent internal state.

This library could be used for this, but the backends do not currently generalize well to 10's or 100's of thousands of streams. One strategy is "multi-plex" events from multiple aggregates on a single stream and have handlers that ignore events that are specific to the target aggregate. The basic trade-off are the number of streams (which may be limited by the backend) and the latency of reading events on a multi-plexed stream.

## Install

```
go get -u github.com/chop-dbhi/eda/...
```

## Backend

This library relies on NATS Streaming as a backend. There are ways of running the server:

- Download a [release binary](https://github.com/nats-io/nats-streaming-server/releases), unzip, and run it.
- Follow the [official quickstart guide](https://nats.io/documentation/streaming/nats-streaming-quickstart/)
- Run the official [Docker container](https://hub.docker.com/_/nats-streaming/)

In any case, the suggested command line options for full durability:

```
$ nats-streaming-server \
  --cluster_id test-cluster \
  --store file \
  --dir data \
  --max_channels 0 \
  --max_subs 0 \
  --max_msgs 0 \
  --max_bytes 0 \
  --max_age 0s \
```

## Get Started

### Connecting to the backend

The first step is to establish a connection to the NATS Streaming server. Below uses the default address and cluster ID. `Connect` also takes a client ID which identifies the connection itself. Be thoughtful of the client ID since it will be used as part of the key for tracking progress in streams for subscribers. It is also added to events published by this connection for traceability.

```go
// Establish a connection to NATS specifiying the server address, the cluster
// ID , and a client ID for this connection.
conn, _ := eda.Connect(
  "nats://localhost:4222",
  "test-cluster",
  "test-client",
)

// Close on exit.
defer conn.Close()
```

### Publishing events

To publish an event, simply use the `Publish` method passing an `Event` value.

```go
id, _ := conn.Publish("subjects", &eda.Event{
  Type: "subject-enrolled",
  Data: eda.JSON(&Subject{
    ID: "3292329",
  }),
})
```

By convention, `Type` should be past tense since it is describing something that already happened. The event `Data` should provide sufficient information on the event for consumers. There are helper functions to encode bytes, strings, JSON, and Protocol Buffer types.

A couple additional fields can be supplied including `Cause` which is an identifier of the upstream event (or something else) that caused this event and `Meta` which is a map of arbitrary key-value pairs for additional information.

Returned is the unique ID of the event and an error if publishing to the server failed.

### Consuming events

The first thing to do is create a `Handler` function that will be used to *handle* events as they are received from the server. You can see the signature of the handler below which includes a `context.Context` value and the received event.

```go
handle := func(ctx context.Context, evt *eda.Event) error {
  switch evt.Type {
  case "subject-enrolled":
    var s Subject
    if err := evt.Data.Decode(&s); err != nil {
      return err
    }

    // Do something with subject.
  }

  return nil
}
```

To start receiving events, a subscription needs to be created which takes the name of the stream to subscribe to, the handler, and subscription options.

```go
sub, _ := conn.Subscribe("subjects", handle, *eda.SubscriptionOptions{
  Backfill: true,
  Durable: true,
  Serial: true,
})
defer sub.Close()
```

In this case, we want to ensure we process events in order even if an error occurs. We want to read the backfill of events in the stream to "catch-up" to the current state, and make the subscription durable so reconnects start where they are left off.

## Learn More

- Check out the [examples directory](./examples) for full examples
- Learn more about [use cases for persistent logs](https://dev.to/byronruth/use-cases-for-persistent-logs-with-nats-streaming)

## License

MIT
