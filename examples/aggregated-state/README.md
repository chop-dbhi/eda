# aggregated-state

Two producers to separate streams at different rates and a consumer that subscribes and aggregates information from both streams and prints out the current state.

## Usage

Start the consumer.

```
go run consumer.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-aggregated-state-consumer \
  -stream test-events \
  -stream2 visit-events
```

Start the first producer of test events.

```
go run producer1.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-aggregated-state-producer1 \
  -stream test-events \
```

Start the second producer of visit events.

```
go run producer2.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-aggregated-state-producer2 \
  -stream visit-events \
```

Note these are the default values, so `go run main.go` is sufficient.
