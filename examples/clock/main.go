package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/chop-dbhi/eda"
)

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
	flag.StringVar(&client, "client", "eda-clock", "Client connection ID.")
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

	// Subscription handler.
	handle := func(ctx context.Context, evt *eda.Event, conn eda.Conn) error {
		log.Println(evt.Type)

		var next string

		switch evt.Type {
		case "tick":
			next = "tock"
		case "tock":
			next = "tick"
		default:
			return nil
		}

		// Delay..
		time.Sleep(time.Second)

		// Publish next event without any data.
		_, err := conn.Publish(stream, &eda.Event{
			Type:  next,
			Cause: evt.ID,
		})

		return err
	}

	// Subscribe to the "clock" stream and use the `handle` function.
	sub, err := conn.Subscribe(
		stream,
		handle,
		nil,
	)
	if err != nil {
		return err
	}
	defer sub.Close()

	// Kick off first event.
	_, err = conn.Publish(stream, &eda.Event{
		Type: "tick",
	})
	if err != nil {
		return err
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	<-sig

	return nil
}
