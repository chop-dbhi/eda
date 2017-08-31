package main

import (
	"context"
	"log"
	"os"

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
	driver := bolt.NewDriver()
	neoConn, err := driver.OpenNeo(os.Getenv("NEO4J_BOLT"))
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
		context.Background(),
		os.Getenv("EDA_ADDR"),
		os.Getenv("EDA_CLUSTER"),
		os.Getenv("EDA_CLIENT_ID"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Subscribe to the target stream.
	_, err = conn.Subscribe(os.Getenv("EDA_STREAM"), handle)
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
