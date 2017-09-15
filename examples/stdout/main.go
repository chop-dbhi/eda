package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/chop-dbhi/eda"
)

func handle(ctx context.Context, evt *eda.Event, conn eda.Conn) error {
	var data interface{}

	switch evt.Data.Type() {
	case "json":
		var m map[string]interface{}
		evt.Data.Decode(&m)
		data = m

		// show format, but no bytes.
	default:
		data = fmt.Sprintf("<< %s >>", evt.Data.Type())
	}

	b, _ := json.Marshal(map[string]interface{}{
		"id":     evt.ID,
		"time":   evt.Time,
		"type":   evt.Type,
		"data":   data,
		"cause":  evt.Cause,
		"client": evt.Client,
	})

	fmt.Fprintln(os.Stdout, string(b))

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		addr    string
		cluster string
		client  string
		stream  string
	)

	flag.StringVar(&addr, "addr", "nats://localhost:4222", "NATS address")
	flag.StringVar(&cluster, "cluster", "test-cluster", "NATS cluster name.")
	flag.StringVar(&client, "client", "eda-stdout", "Client connection ID.")
	flag.StringVar(&stream, "stream", "events", "Stream name.")

	flag.Parse()

	// Establish a client connection to the cluster.
	conn, err := eda.Connect(
		addr,
		cluster,
		client,
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	sub, err := conn.Subscribe(stream, handle, &eda.SubscriptionOptions{
		Durable: false,
	})
	if err != nil {
		return err
	}
	defer sub.Close()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	<-sig

	return nil
}
