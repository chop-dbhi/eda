package eda

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/chop-dbhi/eda/internal/pb"
	"github.com/golang/protobuf/proto"
)

func TestBytesDecodable(t *testing.T) {
	dec := &decodable{
		t: "bytes",
		b: []byte("foo"),
	}

	var v []byte
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(dec.b, v) {
		t.Fatalf("decoded bytes not equal: %v != %v", dec.b, v)
	}
}

func TestRawJSONDecodable(t *testing.T) {
	dec := &decodable{
		t: "json",
		b: []byte(`{"foo": 1}`),
	}

	var v json.RawMessage
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(dec.b, v) {
		t.Fatalf("decoded raw JSON not equal: %v != %v", dec.b, v)
	}
}

func TestJSONDecodable(t *testing.T) {
	x := map[string]int{
		"foo": 1,
	}
	b, _ := json.Marshal(x)

	dec := &decodable{
		t: "json",
		b: b,
	}

	var v map[string]int
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}

	if v["foo"] != x["foo"] {
		t.Fatalf("decoded JSON not equal: %v != %v", x, v)
	}
}

func TestProtoDecodable(t *testing.T) {
	x := &pb.Event{
		Id: "foo",
	}

	b, _ := proto.Marshal(x)

	dec := &decodable{
		t: "proto",
		b: b,
	}

	var v pb.Event
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(x, &v) {
		t.Fatalf("decoded proto not equal: %v != %v", x, v)
	}
}

func TestBytesEncodable(t *testing.T) {
	r := []byte("foo")
	e := Bytes(r)

	// Encode.
	b, err := e.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, r) {
		t.Fatalf("encoded bytes not equal: %v != %v", b, r)
	}

	// Decode.
	var n []byte
	if err = decodeBytes(b, &n); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(n, r) {
		t.Fatalf("decoded bytes not equal: %v != %v", n, r)
	}
}

func TestJSONEncodable(t *testing.T) {
	r := Event{
		ID: "foo",
	}

	e := JSON(&r)

	// Encode.
	b, err := e.Encode()
	if err != nil {
		t.Fatal(err)
	}

	// Decode.
	var n Event
	if err = decodeJSON(b, &n); err != nil {
		t.Fatal(err)
	}

	if r != n {
		t.Fatalf("decoded JSON not equal: %v != %v", n, r)
	}

}

func TestRawJSONEncodable(t *testing.T) {
	r := []byte(`{"foo":1}`)
	e := RawJSON(r)

	// Encode.
	b, err := e.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, r) {
		t.Fatalf("json encoded bytes not equal: %v != %v", b, r)
	}

	// Decode.
	var n json.RawMessage
	if err = decodeJSON(b, &n); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(n, r) {
		t.Fatalf("decoded bytes not equal: %v != %v", n, r)
	}
}

func TestProtoEncodable(t *testing.T) {
	r := pb.Event{
		Id: "foo",
	}
	e := Proto(&r)

	// Encode.
	b, err := e.Encode()
	if err != nil {
		t.Fatal(err)
	}

	// Decode.
	var n pb.Event
	if err := decodeProto(b, &n); err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(&n, &r) {
		t.Fatalf("decoded bytes not equal: %v != %v", &n, &r)
	}
}
