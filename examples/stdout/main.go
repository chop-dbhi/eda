package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/chop-dbhi/eda"
)

var stream = os.Getenv("EDA_STREAM")

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

	_, err = conn.Subscribe(stream, handle)
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
