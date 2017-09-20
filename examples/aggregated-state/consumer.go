package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/chop-dbhi/eda"
)

type Store struct {
	records map[string]*Patient

	// Multiple handlers can update the same record.
	mux sync.Mutex
}

func (s *Store) AddTestRecord(t time.Time, patient, test string, result int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	p, ok := s.records[patient]
	if !ok {
		p = &Patient{
			ID:      patient,
			Created: time.Since(t).Truncate(time.Second).String(),
			Tests:   make(map[string]*Test),
		}

		s.records[patient] = p
	}

	p.LastUpdated = time.Since(t).Truncate(time.Millisecond).String()

	x, ok := p.Tests[test]
	if ok {
		x.Time = time.Since(t).Truncate(time.Second).String()
		d := result - x.Result
		if d > 0 {
			x.Change = fmt.Sprintf("+%d", d)
		} else {
			x.Change = fmt.Sprint(d)
		}
		x.Result = result
	} else {
		p.Tests[test] = &Test{
			Type:   test,
			Time:   time.Since(t).Truncate(time.Second).String(),
			Result: result,
			Change: fmt.Sprint(result),
		}
	}
}

func (s *Store) AddVisit(t time.Time, patient string, visit time.Time) {
	s.mux.Lock()
	defer s.mux.Unlock()

	p, ok := s.records[patient]
	if !ok {
		p = &Patient{
			ID:      patient,
			Created: time.Since(t).Truncate(time.Second).String(),
			Tests:   make(map[string]*Test),
		}

		s.records[patient] = p
	}

	p.LastUpdated = time.Since(t).Truncate(time.Millisecond).String()

	p.LastSeen = fmt.Sprintf("%d days ago", int64(time.Since(visit).Hours()/24))
	p.LatestVisit = visit
	p.NumVisits++
}

type Patient struct {
	ID          string           `json:"id"`
	Created     string           `json:"created"`
	LastUpdated string           `json:"last_updated"`
	Tests       map[string]*Test `json:"tests"`
	LastSeen    string           `json:"last_seen"`
	LatestVisit time.Time        `json:"latest_visit"`
	NumVisits   int              `json:"num_visits"`
}

type Test struct {
	Time   string `json:"time"`
	Type   string `json:"type"`
	Result int    `json:"result"`
	Change string `json:"change"`
}

type PatientTestRecordedEvent struct {
	Patient string
	Type    string
	Result  int
}

type PatientLastVisitedEvent struct {
	Patient   string
	VisitDate time.Time
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
		stream2 string
	)

	flag.StringVar(&addr, "addr", "nats://localhost:4222", "NATS address")
	flag.StringVar(&cluster, "cluster", "test-cluster", "NATS cluster name.")
	flag.StringVar(&client, "client", "eda-aggregated-state-consumer", "Client connection ID.")
	flag.StringVar(&stream, "stream", "test-events", "Test events stream name.")
	flag.StringVar(&stream2, "stream2", "visit-events", "Visit events stream name.")

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

	store := &Store{
		records: make(map[string]*Patient),
	}

	ctx, done := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(time.Second * 1)

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				b, _ := json.MarshalIndent(store.records["pam"], "", " ")
				fmt.Printf(string(b))
			}
		}
	}()

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, os.Kill)

		select {
		case <-sig:
			done()
		}
	}()

	// Subscription handler.
	handle := func(ctx context.Context, evt *eda.Event) error {
		var d PatientTestRecordedEvent
		if err := evt.Data.Decode(&d); err != nil {
			return err
		}

		store.AddTestRecord(evt.Time, d.Patient, d.Type, d.Result)

		return nil
	}

	sub, err := conn.Subscribe(
		stream,
		handle,
		&eda.SubscriptionOptions{
			Backfill: true,
		},
	)
	if err != nil {
		return err
	}
	defer sub.Close()

	// Subscription handler.
	handle2 := func(ctx context.Context, evt *eda.Event) error {
		var d PatientLastVisitedEvent
		if err := evt.Data.Decode(&d); err != nil {
			return err
		}

		store.AddVisit(evt.Time, d.Patient, d.VisitDate)

		return nil
	}

	sub2, err := conn.Subscribe(
		stream2,
		handle2,
		&eda.SubscriptionOptions{
			Backfill: true,
		},
	)
	if err != nil {
		return err
	}
	defer sub2.Close()

	<-ctx.Done()

	return nil
}
