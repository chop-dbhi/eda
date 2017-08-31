# eda

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/chop-dbhi/eda)](https://goreportcard.com/report/github.com/chop-dbhi/eda) [![Build Status](https://travis-ci.org/chop-dbhi/eda.svg?branch=master)](http://travis-ci.org/chop-dbhi/eda) [![GoDoc](https://godoc.org/github.com/chop-dbhi/eda?status.svg)](http://godoc.org/github.com/chop-dbhi/eda)

eda is a library for implementing event-driven architectures. It provides a thin layer on top of backends that support ordered, durable streams with a publish/subscribe interface. The current implementation uses [NATS Streaming](https://github.com/nats-io/nats-streaming-server) as a backend.

## Status

**The library is in an Alpha stage and looking feedback and discussions on API design and scope.**

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
  --cluster_id eda-test \
  --store file \
  --dir data \
  --max_channels 0 \
  --max_subs 0 \
  --max_msgs 0 \
  --max_bytes 0 \
  --max_age 0s \
```

## Example

Below is a fully working example of a "clock" agent that sends two events
"tick" and "tock" on the "clock" stream.

```go
package main

import (
  "context"
  "log"
  "os"
  "time"

  "github.com/chop-dbhi/eda"
)

func main() {
  // Establish a client connection to the cluster. The arguments
  // are the NATS address, cluster id, and client id.
  conn, err := eda.Connect(
    context.Background(),
    "nats://localhost:4222",
    "eda-test",
    "clock-example",
  )
  if err != nil {
    log.Fatal(err)
  }
  defer conn.Close()

  // Subscribe to the "clock" stream and use the `handle` function (below).
  _, err = conn.Subscribe("clock", handle)
  if err != nil {
    log.Fatal(err)
  }

  // Bootstrap by publishing the first event. This takes the stream
  // to publish to, the event type, and any event data.
  _, err = conn.Publish("clock", "tick", nil)
  if err != nil {
    log.Fatal(err)
  }

  // Wait for error or interrupt.
  err = conn.Wait()
  if err != nil {
    log.Fatal(err)
  }
}

// The event handler function receives a context, the event, and a copy
// of the connection in order to publish new events.
func handle(ctx context.Context, evt *eda.Event, conn eda.Conn) error {
  var next string

  // Determine next event.
  switch evt.Type {
  case "tick":
    next = "tock"
  case "tock":
    next = "tick"
  default:
    return nil
  }

  // Delay..
  time.Sleep(time.Second)

  // Publish next event and include ID of causal event.
  _, err := conn.Publish("clock", next, nil, eda.Cause(evt.ID))
  return err
}
```

## License

MIT
