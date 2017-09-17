package eda

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/golang/protobuf/proto"
)

// Encoders are the set of built-in data encMap.
var (
	encMux = &sync.Mutex{}

	encMap = map[string]encoder{
		"bytes": &bytesEncoder{},
		"json":  &jsonEncoder{},
		"proto": &protoEncoder{},
		"nil":   &nilEncoder{},
	}
)

// Encoder is an interface for encoding and decoding native types
// into event data.
type encoder interface {
	Encode(v interface{}) ([]byte, error)
	Decode(b []byte, v interface{}) error
}

// Bytes returns Data that encodes and decodes the raw bytes.
func Bytes(b []byte) Data {
	return &decodable{
		t:   "bytes",
		v:   b,
		enc: encMap["bytes"],
	}
}

// JSON returns Data that encodes and decodes the JSON-encodable value.
func JSON(v interface{}) Data {
	return &decodable{
		t:   "json",
		v:   v,
		enc: encMap["json"],
	}
}

// Proto returns Data that encodes and decodes the proto message.
func Proto(m proto.Message) Data {
	return &decodable{
		t:   "proto",
		v:   m,
		enc: encMap["proto"],
	}
}

type nilEncoder struct{}

func (n *nilEncoder) Type() string {
	return "nil"
}

func (n *nilEncoder) Encode(interface{}) ([]byte, error) {
	return nil, nil
}

func (n *nilEncoder) Decode([]byte, interface{}) error {
	return nil
}

type bytesEncoder struct{}

func (e *bytesEncoder) Encode(v interface{}) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}

	return nil, errors.New("byte slice required")
}

func (e *bytesEncoder) Decode(b []byte, v interface{}) error {
	x, ok := v.(*[]byte)
	if !ok {
		return errors.New("pointer to []byte required")
	}
	*x = b
	return nil
}

type jsonEncoder struct{}

func (e *jsonEncoder) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (e *jsonEncoder) Decode(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

type protoEncoder struct{}

func (e *protoEncoder) Encode(v interface{}) ([]byte, error) {
	if m, ok := v.(proto.Message); ok {
		return proto.Marshal(m)
	}

	return nil, errors.New("proto message required")
}

func (e *protoEncoder) Decode(b []byte, v interface{}) error {
	x, ok := v.(proto.Message)
	if !ok {
		return errors.New("proto.Message required")
	}

	return proto.Unmarshal(b, x)
}

// Data encapsulates a value with a known encoding scheme.
type Data interface {
	// Type returns the encoding type used to encode the data to bytes.
	Type() string

	// Decodes the underlying bytes into the passed value pointer.
	Decode(v interface{}) error

	// Encode encodes the underlying type into bytes.
	Encode() ([]byte, error)
}

// decodable wraps the encoding and raw bytes.
type decodable struct {
	v   interface{}
	b   []byte
	t   string
	e   bool
	enc encoder
}

func (r *decodable) Type() string {
	return r.t
}

// Encode "re-encodes" the bytes without copying.
func (r *decodable) Encode() ([]byte, error) {
	if r.e {
		return r.b, nil
	}

	if r.enc != nil {
		return r.enc.Encode(r.v)
	}

	return nil, errors.New("unknown encoding: " + r.t)
}

func (r *decodable) Decode(v interface{}) error {
	if r.e {
		if r.enc != nil {
			return r.enc.Decode(r.b, v)
		}

		return errors.New("unknown encoding: " + r.t)
	}

	return errors.New("cannot decode non-encoded data")
}
