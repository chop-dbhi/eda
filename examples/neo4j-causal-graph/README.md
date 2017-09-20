# neo4j-causal-graph

Consumer that creates nodes in Neo4j and causal relationships (edges) between nodes based on the `Event.Cause` ID.

## Usage

This requires Neo4j 3.0+ to utilize the Bolt protocol.

```
go run main.go \
  -neo4j bolt://localhost:7687 \
  -addr nats://localhost:4222 \
  -cluster test-cluster \
  -client eda-neo4j-causal-graph \
  -stream events
```

Note these are the default values, so `go run main.go` is sufficient.
