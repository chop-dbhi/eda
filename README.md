# eda

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/chop-dbhi/eda)](https://goreportcard.com/report/github.com/chop-dbhi/eda) [![Build Status](https://travis-ci.org/chop-dbhi/eda.svg?branch=master)](http://travis-ci.org/chop-dbhi/eda) [![GoDoc](https://godoc.org/github.com/chop-dbhi/eda?status.svg)](http://godoc.org/github.com/chop-dbhi/eda)

eda is a library for implementing event-driven architectures. It provides a set of concrete types that are *envelopes* for common messages types, including `Message`, `Command`, `Reply`, and `Event`. In addition a `Data` type is provided which provides an easy way to wrap message data (your domain-specific types) and embed them within the envelopes. All types have built-in support for marshaling and unmarshaling so they can be transmitted over the wire using a transport of your choice.

## Status

**The library is in an Alpha stage and looking feedback and discussions on API design and scope. Check out the [issues](https://github.com/chop-dbhi/eda/issues) for topics.**

## Install

Requires Go 1.9+

```
go get -u github.com/chop-dbhi/eda
```

## Quickstart

Below is a type that represents a domain event an a study recruitment application.

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

This event only contains information relevant to the domain and not boilerplate event metadata such as the time, unique ID, etc. This is handled by the `eda.Event` type that wraps the domain event treating it as the event data.

```go
e := eda.Event{
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

The use of `eda.JSON` will convert the domain event into a `eda.Data` type with a JSON encoding. Additional built-in codecs including `eda.Proto`, `eda.String`, and `eda.Bytes`.

When the event is marshalled to bytes, the event will be internally marshalled to JSON. A call to `Marshal` will automatically generate a unique ID and set the current time on the event. These values can be set ahead of time if more control is needed.

```go
// Marshal to bytes.
b, err := e.Marshal()
```

Internally marshalling is done with a Protocol Buffer message type. This is for efficiency, but also allows for cross-language support.

At this point the bytes can be sent over the wire using a built-in [transport](#transports) or one of your choice. To manually unmarshal, simply do the following:

```go
var e eda.Event
err := e.Unmarshal(b)
```

The various properties of the event can be accessed such as `Type`, `Time`, `ID`, etc. To access the original domain event simply initialize a value of the corresponding type and decode it from the `Data` field.

```go
var de ScreeningAppointmentScheduled
err := e.Data.Decode(&de)
```

This will decode the original domain event into the new value. In practice, a common pattern is to switch on the `Type` field to select which concrete type to decode.

```go
// Common pattern to matching the correct type.
switch e.Type {
case "ScreeningAppointmentScheduled":
  var de ScreeningAppointmentScheduled
  err := e.Data.Decode(&de)

// handle other types..
}
```

For more information on the additional fields, check out the [docs](https://godoc.org/github.com/chop-dbhi/eda).

## Use Case

The primary use case this library is being designed to support are applications involving "domain events". That is, these events carry information about something that occurred in a domain model that must be made available for other consumers.

One application of this is as a building block for systems using CQRS pattern where events produced on the write side (a result of handling a command) need to get published so the read side can consume and update their internal indexes.

Another related use case is Event Sourcing which are generally spoken of in the context of an "aggregate". The pattern requires each aggregate instance to maintain it's own stream of events acting as an internal changelog. This stream is generally "private" from other consumers and requires having a single handler to apply events in order to maintain a consistent internal state.

This library could be used for this, but the backends do not currently generalize well to 10's or 100's of thousands of streams. One strategy is "multi-plex" events from multiple aggregates on a single stream and have handlers that ignore events that are specific to the target aggregate. The basic trade-off are the number of streams (which may be limited by the backend) and the latency of reading events on a multi-plexed stream.

## License

MIT
