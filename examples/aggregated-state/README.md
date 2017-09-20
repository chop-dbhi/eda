# aggregated-state

Two producers to separate streams at different rates and a consumer that subscribes and aggregates information from both streams and prints out the current state.

## Usage

Start the consumer.

```
go run consumer/main.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-aggregated-state-consumer \
  -stream test-events \
  -stream2 visit-events
```

Start the first producer of test events.

```
go run producer-test-events/main.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-aggregated-state-producer-test-events \
  -stream test-events \
```

Start the second producer of visit events.

```
go run producer-visit-events/main.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-aggregated-state-producer-visit-events \
  -stream visit-events \
```

Note these are the default values, so `go run main.go` is sufficient.
