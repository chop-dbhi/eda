// The codec package implements codec for encoding and decoding message data.
package codec

import (
	"encoding"
	"encoding/json"
	"errors"

	"github.com/golang/protobuf/proto"
)

var (
	Bytes  = &bytesCodec{}
	Binary = &binaryCodec{}
	String = &stringCodec{}
	JSON   = &jsonCodec{}
	Proto  = &protoCodec{}

	codecs = map[string]Codec{
		"bytes":  Bytes,
		"binary": Binary,
		"string": String,
		"json":   JSON,
		"proto":  Proto,
	}
)

// Codec is an interface for encoding and decoding native types into bytes.
type Codec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(b []byte, v interface{}) error
}

// Register registers a codec.
func Register(name string, codec Codec) {
	codecs[name] = codec
}

// Get gets a codec by name.
func Get(name string) (Codec, bool) {
	c, ok := codecs[name]
	return c, ok
}

type binaryCodec struct{}

func (e *binaryCodec) Marshal(v interface{}) ([]byte, error) {
	if m, ok := v.(encoding.BinaryMarshaler); ok {
		return m.MarshalBinary()
	}

	return nil, errors.New("encoding.BinaryMarshaler required")
}

func (e *binaryCodec) Unmarshal(b []byte, v interface{}) error {
	if m, ok := v.(encoding.BinaryUnmarshaler); ok {
		return m.UnmarshalBinary(b)
	}

	return errors.New("encoding.BinaryUnmarshaler required")
}

type bytesCodec struct{}

func (e *bytesCodec) Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}

	return nil, errors.New("byte slice required")
}

func (e *bytesCodec) Unmarshal(b []byte, v interface{}) error {
	x, ok := v.(*[]byte)
	if !ok {
		return errors.New("pointer to []byte required")
	}
	*x = b
	return nil
}

type stringCodec struct{}

func (e *stringCodec) Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.(string); ok {
		return []byte(b), nil
	}

	return nil, errors.New("string required")
}

func (e *stringCodec) Unmarshal(b []byte, v interface{}) error {
	x, ok := v.(*string)
	if !ok {
		return errors.New("pointer to string required")
	}
	*x = string(b)
	return nil
}

type jsonCodec struct{}

func (e *jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (e *jsonCodec) Unmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

type protoCodec struct{}

func (e *protoCodec) Marshal(v interface{}) ([]byte, error) {
	if m, ok := v.(proto.Message); ok {
		return proto.Marshal(m)
	}

	return nil, errors.New("proto message required")
}

func (e *protoCodec) Unmarshal(b []byte, v interface{}) error {
	x, ok := v.(proto.Message)
	if !ok {
		return errors.New("proto.Message required")
	}

	return proto.Unmarshal(b, x)
}
