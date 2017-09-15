/*
The eda package is a library for implementing event-driven architectures. It provides a thin layer on top of backends that support ordered streams with a publish/subscribe interface. The current implementation uses NATS Streaming: https://github.com/nats-io/nats-streaming-server, but additional backends could be supported.

Use Case

The primary use case this library is being designed to support are applications involving "domain events". That is, these events carry information about something that occurred in a domain model that must be made available for other consumers.

One application of this is as a building block for systems using CQRS pattern where events produced on the write side (a result of handling a command) need to get published so the read side can consume and update their internal indexes.

Another related use case is Event Sourcing which are generally spoken of in the context of an "aggregate". The pattern requires each aggregate instance to maintain it's own stream of events acting as an internal changelog. This stream is generally "private" from other consumers and requires having a single handler to apply events in order to maintain a consistent internal state.

This library could be used for this, but the backends do not currently generalize well to 10's or 100's of thousands of streams. One strategy is "multi-plex" events from multiple aggregates on a single stream and have handlers that ignore events that are specific to the target aggregate. The basic trade-off are the number of streams (which may be limited by the backend) and the latency of reading events on a multi-plexed stream.
*/
package eda
