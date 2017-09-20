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

type PatientLastVisitedEvent struct {
	Patient   string
	VisitDate time.Time
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

func randomEvent() *PatientLastVisitedEvent {
	p := patients[rand.Intn(len(patients))]

	return &PatientLastVisitedEvent{
		Patient:   p,
		VisitDate: time.Now().Add(-24 * time.Hour * time.Duration(rand.Intn(30))),
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
	flag.StringVar(&client, "client", "eda-aggregated-state-producer-visit-events", "Client connection ID.")
	flag.StringVar(&stream, "stream", "visit-events", "Event stream name.")

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
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

			_, err = conn.Publish(stream, &eda.Event{
				Type: "patient-last-visited",
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
