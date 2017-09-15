package main

import (
	"context"
	"flag"
	"log"

	"github.com/chop-dbhi/eda"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
)

var (
	// Idempotent commmands.
	startupQueries = []string{
		`CREATE CONSTRAINT ON (e:Event) ASSERT e.id IS UNIQUE`,
		`CREATE INDEX ON :Event(time)`,
		`CREATE INDEX ON :Event(type)`,
	}

	createNode = `
		MERGE (n:Event {id: {id}})
		ON CREATE SET
			n.type = {type},
			n.time = {time},
			n.client = {client},
			n.encoding = {encoding}

		ON MATCH SET
			n.type = {type},
			n.time = {time},
			n.client = {client},
			n.encoding = {encoding}
	`

	createEdge = `
		MATCH (s:Event {id: {src}}), (d:Event {id: {dst}})
		MERGE (s)-[:caused]->(d)
	`
)

func main() {
	var (
		addr      string
		cluster   string
		client    string
		stream    string
		neo4jBolt string
	)

	flag.StringVar(&addr, "addr", "nats://localhost:4222", "NATS address")
	flag.StringVar(&cluster, "cluster", "test-cluster", "NATS cluster name.")
	flag.StringVar(&client, "client", "neo4j-causal-graph", "Client connection ID.")
	flag.StringVar(&stream, "stream", "test", "Stream name.")
	flag.StringVar(&neo4jBolt, "neo4j.bolt", "", "Neo4j bolt address.")

	flag.Parse()

	driver := bolt.NewDriver()
	neoConn, err := driver.OpenNeo(neo4jBolt)
	if err != nil {
		log.Fatal(err)
	}
	defer neoConn.Close()

	for _, q := range startupQueries {
		if _, err := neoConn.ExecNeo(q, nil); err != nil {
			log.Fatal(err)
		}
	}

	handle := func(ctx context.Context, evt *eda.Event, _ eda.Conn) error {
		log.Printf("received event: %s", evt.Type)
		_, err = neoConn.ExecNeo(createNode, map[string]interface{}{
			"id":       evt.ID,
			"type":     evt.Type,
			"time":     evt.Time,
			"client":   evt.Client,
			"encoding": evt.Data.Type(),
		})
		if err != nil {
			return err
		}

		// No cause means this is an edge event.
		if evt.Cause == "" {
			return nil
		}

		_, err = neoConn.ExecNeo(createEdge, map[string]interface{}{
			"src": evt.Cause,
			"dst": evt.ID,
		})

		return err
	}

	// Establish a client connection to the cluster.
	conn, err := eda.Connect(
		addr,
		cluster,
		client,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Subscribe to the target stream.
	_, err = conn.Subscribe(stream, handle, &eda.SubscriptionOptions{
		Durable:  true,
		Backfill: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
