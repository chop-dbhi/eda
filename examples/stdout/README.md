# stdout

Consumer that writes events in a JSON encoding to stdout. If the event data is not JSON-encoded, it will simply write out a placeholder, e.g. `<< proto >>`.

## Usage

```
go run main.go \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-stdout \
  -stream events
```

Note these are the default values, so `go run main.go` is sufficient.
