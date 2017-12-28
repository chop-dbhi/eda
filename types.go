package eda

import (
	"encoding"
	"encoding/json"
	"errors"
	"time"

	"github.com/chop-dbhi/eda/codec"
	"github.com/chop-dbhi/eda/internal/pb"
	gpack "github.com/glycerine/greenpack/msgp"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nuid"
	"github.com/tinylib/msgp/msgp"
)

var (
	// NewID returns a new unique ID used for the messages.
	// This function is used when a message is marshaled
	// and does not have an ID set.
	NewID func() string = nuid.Next
)

// Message represents a general message type.
type Message struct {
	// ID is the unique ID of the message.
	ID string

	// Type is the message type.
	Type string

	// Time when the message was sent by the client.
	Time time.Time

	// Data is the encoded message data.
	Data *Data

	// Meta is encoded meta data for the message.
	Meta *Data

	// Correlation is an identifier that correlates related messages.
	// This is often used for *sagas* or general tracing.
	CorrelationID string

	// Causal is the ID of a message that cause this message to be sent.
	// This is used in the context of events and commands.
	CausalID string

	// Topic is the topic the message was published to. Transport
	// implementations can set this value for consumers.
	Topic string

	// Time the message was acknowledged by the transport. Transport
	// implementations that support ACKs can set this value for consumers.
	AckTime time.Time
}

func (m *Message) setDefaults() {
	// Generate and set the ID if not set.
	if m.ID == "" {
		m.ID = NewID()
	}

	// Set the event time if not set.
	if m.Time.IsZero() {
		m.Time = time.Now()
	}
}

// MarshalJSON marshals the message into JSON. This can be used as
// a human-readable alternative for debugging and other read-only
// use cases. Sets the message ID and time if not set.
func (m *Message) MarshalJSON() ([]byte, error) {
	m.setDefaults()

	x := map[string]interface{}{
		"id":   m.ID,
		"type": m.Type,
		"time": m.Time.UTC(),
	}

	if m.CorrelationID != "" {
		x["correlation_id"] = m.CorrelationID
	}

	if m.CausalID != "" {
		x["causal_id"] = m.CausalID
	}

	if m.Topic != "" {
		x["topic"] = m.Topic
	}

	if !m.AckTime.IsZero() {
		x["ack_time"] = m.AckTime.UTC()
	}

	// Marshal the data and meta fields into the internal proto bytes.
	if m.Data != nil {
		x["data"] = m.Data
	}

	if m.Meta != nil {
		x["meta"] = m.Meta
	}

	return json.Marshal(x)
}

// Marshal marshals the message into proto bytes.
// Sets the message ID and time if not set.
func (m *Message) Marshal() ([]byte, error) {
	m.setDefaults()

	x := &pb.Message{
		Id:            m.ID,
		Type:          m.Type,
		Time:          m.Time.UnixNano(),
		CorrelationId: m.CorrelationID,
		CausalId:      m.CausalID,
	}

	// Marshal the data and meta fields into the internal proto bytes.
	if m.Data != nil {
		b, err := m.Data.Marshal()
		if err != nil {
			return nil, err
		}
		x.Data = b
	}

	if m.Meta != nil {
		b, err := m.Meta.Marshal()
		if err != nil {
			return nil, err
		}
		x.Meta = b
	}

	return proto.Marshal(x)
}

// Unmarshal unmarshals the proto bytes into the message.
func (m *Message) Unmarshal(b []byte) error {
	var x pb.Message

	// Message sent on stream that is not a protobuf format.
	if err := proto.Unmarshal(b, &x); err != nil {
		return err
	}

	if x.Data != nil {
		var data Data
		if err := data.Unmarshal(x.Data); err != nil {
			return err
		}
		m.Data = &data
	}

	if x.Meta != nil {
		var meta Data
		if err := meta.Unmarshal(x.Meta); err != nil {
			return err
		}
		m.Meta = &meta
	}

	m.ID = x.Id
	m.Time = time.Unix(0, x.Time)
	m.Type = x.Type
	m.CorrelationID = x.CorrelationId
	m.CausalID = x.CausalId

	return nil
}

// Reply corresponds to a reply to a command.
type Reply struct {
	// Status code of the reply.
	Code Code

	// Status message of the reply.
	Message string

	// Domain-specific data included in the reply.
	Data *Data

	// Domain-specific metadata in the reply.
	Meta *Data
}

// MarshalJSON marshals the reply to JSON bytes.
func (r *Reply) MarshalJSON() ([]byte, error) {
	x := map[string]interface{}{
		"code": r.Code,
	}

	if r.Message != "" {
		x["message"] = r.Message
	}

	if r.Data != nil {
		x["data"] = r.Data
	}

	if r.Meta != nil {
		x["meta"] = r.Meta
	}

	return json.Marshal(x)
}

// Marshal marshals the reply to proto bytes.
func (r *Reply) Marshal() ([]byte, error) {
	x := &pb.Reply{
		Code: int32(r.Code),
		Msg:  r.Message,
	}

	if r.Data != nil {
		b, err := r.Data.Marshal()
		if err != nil {
			return nil, err
		}
		x.Data = b
	}

	if r.Meta != nil {
		b, err := r.Meta.Marshal()
		if err != nil {
			return nil, err
		}
		x.Meta = b
	}

	return proto.Marshal(x)
}

// Unmarshal unmarshal proto bytes into the reply.
func (r *Reply) Unmarshal(b []byte) error {
	var x pb.Reply
	// Message sent on stream that is not a protobuf format.
	if err := proto.Unmarshal(b, &x); err != nil {
		return err
	}

	var data Data
	if err := data.Unmarshal(x.Data); err != nil {
		return err
	}

	r.Code = Code(x.Code)
	r.Message = x.Msg
	r.Data = &data

	return nil
}

// Data encapsulates a value with a known encoding scheme.
type Data struct {
	// Encoding is the byte encoding of the data.
	Encoding string

	// Schema of the data payload.
	Schema string

	// Value is value to be encoded.
	value interface{}

	// Bytes are the encoded value from and set from an
	// unmarshaled value.
	bytes []byte
}

// Set sets a new value and clears any internally cache bytes.
func (d *Data) Set(v interface{}) {
	d.value = v
	d.bytes = nil
}

// MarshalJSON marshals the data to JSON bytes.
func (d *Data) MarshalJSON() ([]byte, error) {
	x := map[string]interface{}{
		"encoding": d.Encoding,
	}

	if d.Schema != "" {
		x["schema"] = d.Schema
	}

	// Use value if set, otherwise attempt to convert bytes to JSON.
	if d.value != nil {
		x["value"] = d.value
	} else if d.bytes != nil {
		switch d.Encoding {
		case "json":
			x["value"] = json.RawMessage(d.bytes)

		case "string":
			var s string
			if err := d.Decode(&s); err != nil {
				return nil, err
			}

			x["value"] = s

		default:
			x["value"] = d.bytes
		}
	}

	return json.Marshal(x)
}

// Marshal marshals the data into proto bytes.
func (d *Data) Marshal() ([]byte, error) {
	// No content to the data.
	if d.value == nil && d.bytes == nil {
		return nil, nil
	}

	// Encoding is not set.
	if d.Encoding == "" {
		return nil, errors.New("no encoding specified")
	}

	x := &pb.Data{
		Schema:   d.Schema,
		Encoding: d.Encoding,
	}

	// Use existing bytes or marshal the new value.
	if d.bytes != nil {
		x.Data = d.bytes
	} else {
		c, ok := codec.Get(d.Encoding)
		if !ok {
			return nil, errors.New("no codec for " + d.Encoding)
		}

		b, err := c.Marshal(d.value)
		if err != nil {
			return nil, err
		}

		x.Data = b
	}

	return proto.Marshal(x)
}

// Unmarshal unmarshals proto bytes into the value.
func (d *Data) Unmarshal(b []byte) error {
	var x pb.Data
	if err := proto.Unmarshal(b, &x); err != nil {
		return err
	}

	d.Schema = x.Schema
	d.Encoding = x.Encoding

	// Store encoded bytes for decoding.
	d.bytes = x.Data

	return nil
}

// Decode decodes the data into the passed value.
func (d *Data) Decode(v interface{}) error {
	if d.bytes == nil {
		return errors.New("no data to decode")
	}

	c, ok := codec.Get(d.Encoding)
	if !ok {
		return errors.New("no codec for " + d.Encoding)
	}

	return c.Unmarshal(d.bytes, v)
}

// Binary returns Data that encodes the binary marshaler.
func Binary(m encoding.BinaryMarshaler) *Data {
	return &Data{
		Encoding: "binary",
		value:    m,
	}
}

// Bytes returns Data that encodes raw bytes.
func Bytes(b []byte) *Data {
	return &Data{
		Encoding: "bytes",
		value:    b,
	}
}

// String returns Data that encodes the string.
func String(s string) *Data {
	return &Data{
		Encoding: "string",
		value:    s,
	}
}

// JSON returns Data that encodes the JSON-encodable value.
func JSON(v interface{}) *Data {
	return &Data{
		Encoding: "json",
		value:    v,
	}
}

// Proto returns Data that encodes the proto message.
func Proto(m proto.Message) *Data {
	return &Data{
		Encoding: "proto",
		value:    m,
	}
}

// Msgpack returns Data that encodes the msgpack message.
func Msgpack(m msgp.Encodable) *Data {
	return &Data{
		Encoding: "msgpack",
		value:    m,
	}

}

// Greenpack returns Data that encodes the greenpack message.
func Greenpack(m gpack.Encodable) *Data {
	return &Data{
		Encoding: "greenpack",
		value:    m,
	}
}
