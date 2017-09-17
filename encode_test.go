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
		t:   "bytes",
		b:   []byte("foo"),
		e:   true,
		enc: encMap["bytes"],
	}

	var v []byte
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(dec.b, v) {
		t.Fatalf("decoded bytes not equal: %v != %v", dec.b, v)
	}
}

func TestStringDecodable(t *testing.T) {
	dec := &decodable{
		t:   "string",
		b:   []byte("foo"),
		e:   true,
		enc: encMap["string"],
	}

	var v string
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}

	if string(dec.b) != v {
		t.Fatalf("decoded string not equal: %v != %v", dec.b, v)
	}
}

func TestJSONDecodable(t *testing.T) {
	x := map[string]int{
		"foo": 1,
	}
	b, _ := json.Marshal(x)

	dec := &decodable{
		t:   "json",
		b:   b,
		e:   true,
		enc: encMap["json"],
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
		t:   "proto",
		b:   b,
		e:   true,
		enc: encMap["proto"],
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
	if err = (&bytesEncoder{}).Decode(b, &n); err != nil {
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
	if err = (&jsonEncoder{}).Decode(b, &n); err != nil {
		t.Fatal(err)
	}

	if r.ID != n.ID {
		t.Fatalf("decoded JSON not equal: %v != %v", n, r)
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
	if err := (&protoEncoder{}).Decode(b, &n); err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(&n, &r) {
		t.Fatalf("decoded bytes not equal: %v != %v", &n, &r)
	}
}
