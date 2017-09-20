# clock

Consumer and producer that models "tick" and "tock" as events.

## Usage

```
go run main.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-clock \
  -stream events
```

Note these are the default values, so `go run main.go` is sufficient.
