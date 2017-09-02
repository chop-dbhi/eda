package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chop-dbhi/eda"
)

var stream = os.Getenv("EDA_STREAM")

func main() {
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

	// Subscribe to the "clock" stream and use the `handle` function.
	_, err = conn.Subscribe(
		stream,
		handle,
		eda.Durable(false),
		eda.StartNew(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Kick off first event.
	_, err = conn.Publish(stream, "tick", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for error or interrupt.
	err = conn.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func handle(ctx context.Context, evt *eda.Event, conn eda.Conn) error {
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
	_, err := conn.Publish(stream, next, nil, eda.Cause(evt.ID))
	return err
}
