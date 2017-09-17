package eda

import (
	"context"
	"flag"
	"testing"
)

var (
	addr    string
	cluster string
	client  string
	stream  string
)

func init() {
	flag.StringVar(&addr, "addr", "nats://localhost:4222", "NATS address")
	flag.StringVar(&cluster, "cluster", "test-cluster", "NATS cluster name.")
	flag.StringVar(&client, "client", "test-client", "Client connection ID.")
	flag.StringVar(&stream, "stream", "test-stream", "Stream name.")
}

func TestSubscribe(t *testing.T) {
	conn, err := Connect(addr, cluster, client)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	handle := func(ctx context.Context, evt *Event) error {
		return nil
	}

	sub, err := conn.Subscribe(stream, handle, nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer sub.Close()
}
