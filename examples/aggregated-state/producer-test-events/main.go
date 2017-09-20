package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/chop-dbhi/eda"
)

type PatientTestRecordedEvent struct {
	Patient string
	Type    string
	Result  int
}

var patients = []string{
	"joe",
	"pam",
	"sue",
	"jane",
	"bob",
	"jeff",
	"jon",
}

var testTypes = []string{
	"left, air",
	"right, air",
	"left, bone",
	"right, bone",
	"both",
}

var testResults = []int{
	0, 5, 10, 15, 20, 25, 30, 35, 40, 45,
	50, 55, 60, 65, 70, 75, 80, 85, 90,
	95, 100, 105, 110, 115, 120,
}

func randomEvent() *PatientTestRecordedEvent {
	p := patients[rand.Intn(len(patients))]
	t := testTypes[rand.Intn(len(testTypes))]
	r := testResults[rand.Intn(len(testResults))]

	return &PatientTestRecordedEvent{
		Patient: p,
		Type:    t,
		Result:  r,
	}
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
	flag.StringVar(&client, "client", "eda-aggregated-state-producer-test-events", "Client connection ID.")
	flag.StringVar(&stream, "stream", "test-events", "Event stream name.")

	flag.Parse()

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

	ctx, done := context.WithCancel(context.Background())

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, os.Kill)

		select {
		case <-sig:
			done()
		}
	}()

	var count int

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			// Simulate delay..
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

			_, err = conn.Publish(stream, &eda.Event{
				Type: "patient-test-recorded",
				Data: eda.JSON(randomEvent()),
			})
			if err != nil {
				return err
			}

			count++

			fmt.Printf("\rpublished %d events", count)
		}
	}

	return nil
}
