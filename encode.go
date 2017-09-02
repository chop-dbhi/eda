package eda

import (
	"encoding/json"
	"errors"

	"github.com/golang/protobuf/proto"
)

// Decoders are the set of decoders
var Decoders = map[string]Decoder{
	"bytes": DecodeBytes,
	"json":  DecodeJSON,
	"proto": DecodeProto,
}

// Decoder wraps bytes into a Decodable for deserialization.
type Decoder func(b []byte, v interface{}) error

// Encodable encapsulates encodable data.
type Encodable interface {
	// Type returns the type of encodable.
	Type() string

	// Encode encodes the underlying type into bytes.
	Encode() ([]byte, error)
}

// Decodable encapsulates bytes that can be decoded.
type Decodable interface {
	// Type returns the encoding type of the bytes.
	Type() string

	// Decodes the underlying bytes into the passed value pointer.
	Decode(v interface{}) error

	// Encode encodes the underlying type into bytes.
	Encode() ([]byte, error)
}

// decodable wraps the encoding and raw bytes.
type decodable struct {
	b []byte
	t string
}

func (r *decodable) Type() string {
	return r.t
}

// Encode "re-encodes" the bytes without copying.
func (r *decodable) Encode() ([]byte, error) {
	return r.b, nil
}

func (r *decodable) Decode(v interface{}) error {
	if decoder, ok := Decoders[r.t]; ok {
		return decoder(r.b, v)
	}

	return errors.New("unknown encoding: " + r.t)
}

func DecodeBytes(b []byte, v interface{}) error {
	x, ok := v.(*[]byte)
	if !ok {
		return errors.New("pointer to []byte required")
	}
	*x = b
	return nil
}

func DecodeJSON(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

func DecodeProto(b []byte, v interface{}) error {
	x, ok := v.(proto.Message)
	if !ok {
		return errors.New("proto.Message required")
	}

	return proto.Unmarshal(b, x)
}

type encodableBytes []byte

func (encodableBytes) Type() string {
	return "bytes"
}

func (e encodableBytes) Encode() ([]byte, error) {
	return e, nil
}

// Bytes wraps a byte slice as an encodable.
func Bytes(b []byte) Encodable {
	return encodableBytes(b)
}

type encodableJSON struct {
	v interface{}
}

func (*encodableJSON) Type() string {
	return "json"
}

func (e *encodableJSON) Encode() ([]byte, error) {
	return json.Marshal(e.v)
}

// JSON wraps a value intended to be JSON-encoded as an encodable.
func JSON(v interface{}) Encodable {
	return &encodableJSON{v}
}

type encodableProto struct {
	v proto.Message
}

func (*encodableProto) Type() string {
	return "proto"
}

func (e *encodableProto) Encode() ([]byte, error) {
	return proto.Marshal(e.v)
}

// Proto wraps a protobuf message as an encodable.
func Proto(msg proto.Message) Encodable {
	return &encodableProto{msg}
}
