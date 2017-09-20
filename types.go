package eda

import (
	"encoding"
	"errors"
	"time"

	"google.golang.org/grpc/status"

	"github.com/chop-dbhi/eda/codec"
	"github.com/chop-dbhi/eda/internal/pb"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nuid"
)

var (
	// GenId is the ID generator for messages. This can be overridden to
	// use a different ID generation method.
	GenId IdGen = GenNatsId

	valueUnset = struct{}{}
)

// GenNatsId generates a new ID using the NATS nuid library.
func GenNatsId() string {
	return nuid.Next()
}

// IdGen is a type for generating unique message IDs.
type IdGen func() string

// Message represents a general message type.
type Message struct {
	// ID is the unique ID of the message.
	ID string `json:"id"`

	// Type is the message type.
	Type string `json:"type"`

	// Time when the meessage was sent by the client.
	Time time.Time `json:"time"`

	// Data is the encoded message data.
	Data *Data `json:"data,omitempty"`

	// Meta is encoded meta data for the message.
	Meta *Data `json:"meta,omitempty"`
}

// Marshal marshals the message into an internal proto bytes.
// If not set, the message ID and current time is set.
func (m *Message) Marshal() ([]byte, error) {
	var (
		err        error
		data, meta []byte
	)

	// Marshal the data and meta fields into the internal proto bytes.
	if m.Data != nil {
		if data, err = m.Data.Marshal(); err != nil {
			return nil, err
		}
	}

	if m.Meta != nil {
		if meta, err = m.Meta.Marshal(); err != nil {
			return nil, err
		}
	}

	// Generate and set the ID if not set.
	if m.ID == "" {
		m.ID = GenId()
	}

	// Set the event time if not set.
	if m.Time.IsZero() {
		m.Time = time.Now()
	}

	return proto.Marshal(&pb.Message{
		Id:   m.ID,
		Type: m.Type,
		Time: m.Time.UnixNano(),
		Data: data,
		Meta: meta,
	})
}

// Unmarshals the proto bytes into this type.
func (m *Message) Unmarshal(b []byte) error {
	var x pb.Message

	// Message sent on stream that is not a protobuf format.
	if err := proto.Unmarshal(b, &x); err != nil {
		return err
	}

	var data, meta Data
	if err := data.Unmarshal(x.Data); err != nil {
		return err
	}

	if err := meta.Unmarshal(x.Meta); err != nil {
		return err
	}

	m.ID = x.Id
	m.Time = time.Unix(0, x.Time)
	m.Type = x.Type
	m.Data = &data
	m.Meta = &meta

	return nil
}

// Command corresponds to a command sent to a command handler.
type Command struct {
	// ID is the unique ID of the command.
	ID string `json:"id"`

	// Type is the command type.
	Type string `json:"type"`

	// Time when the command was sent by the client.
	Time time.Time `json:"time"`

	// Aggregate is the ID of the aggregate this command applies to.
	Aggregate string `json:"aggregate"`

	// Data is the encoded command data.
	Data *Data `json:"data,omitempty"`

	// Meta is encoded meta data for the command.
	Meta *Data `json:"meta,omitempty"`
}

// PrepareCommand implements the CommandModel interface.
func (c *Command) PrepareCommand() *Command {
	return c
}

func (c *Command) Unmarshal(b []byte) error {
	var x pb.Command
	// Message sent on stream that is not a protobuf format.
	if err := proto.Unmarshal(b, &x); err != nil {
		return err
	}

	var data, meta Data
	if err := data.Unmarshal(x.Msg.Data); err != nil {
		return err
	}

	if err := meta.Unmarshal(x.Msg.Meta); err != nil {
		return err
	}

	c.ID = x.Msg.Id
	c.Aggregate = x.Aggregate
	c.Time = time.Unix(0, x.Msg.Time)
	c.Type = x.Msg.Type
	c.Data = &data
	c.Meta = &meta

	return nil
}

func (c *Command) Marshal() ([]byte, error) {
	var (
		err        error
		data, meta []byte
	)

	if c.Data != nil {
		if data, err = c.Data.Marshal(); err != nil {
			return nil, err
		}
	}

	if c.Meta != nil {
		if meta, err = c.Meta.Marshal(); err != nil {
			return nil, err
		}
	}

	if c.ID == "" {
		c.ID = GenId()
	}

	// Add command time if not set.
	if c.Time.IsZero() {
		c.Time = time.Now()
	}

	return proto.Marshal(&pb.Command{
		Aggregate: c.Aggregate,
		Msg: &pb.Message{
			Id:   c.ID,
			Type: c.Type,
			Time: c.Time.UnixNano(),
			Data: data,
			Meta: meta,
		},
	})
}

// Reply corresponds to a reply to a command.
type Reply struct {
	// Status is the status code and message of the reply.
	Status *status.Status `json:"status"`

	// Reply data.
	Data *Data `json:"data,omitempty"`
}

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

	r.Status = x.Status
	r.Data = &data

	return nil
}

func (r *Reply) Marshal() ([]byte, error) {
	var (
		err  error
		data []byte
	)

	if r.Data != nil {
		if data, err = r.Data.Marshal(); err != nil {
			return nil, err
		}
	}

	return proto.Marshal(&pb.Reply{
		Status: r.Status,
		Data:   data,
	})
}

// Event is a type for encoding and decoding an event.
type Event struct {
	// ID is the unique ID of the event.
	ID string `json:"id"`

	// Topic is the topic the event was published on.
	Topic string `json:"topic"`

	// Type is the event type.
	Type string `json:"type"`

	// Time when the event was published by the client.
	Time time.Time `json:"time"`

	// Time the event was acknowledged by the server.
	AckTime time.Time `json:"ack_time"`

	// Aggregate is the ID of the aggregate this event pertains to.
	Aggregate string `json:"aggregate"`

	// Causes are the set of IDs that caused this event.
	Causes []string `json:"causes,omitempty"`

	// Data is the encoded event data.
	Data *Data `json:"data,omitempty"`

	// Meta is arbitrary meta data for the event itself.
	Meta *Data `json:"meta,omitempty"`
}

func (e *Event) Unmarshal(b []byte) error {
	var x pb.Event
	// Message sent on stream that is not a protobuf format.
	if err := proto.Unmarshal(b, &x); err != nil {
		return err
	}

	var data, meta Data
	if err := data.Unmarshal(x.Msg.Data); err != nil {
		return err
	}

	if err := meta.Unmarshal(x.Msg.Meta); err != nil {
		return err
	}

	e.ID = x.Msg.Id
	e.Time = time.Unix(0, x.Msg.Time)
	e.Type = x.Msg.Type
	e.Causes = x.Causes
	e.Aggregate = x.Aggregate
	e.Data = &data
	e.Meta = &meta

	return nil
}

func (e *Event) Marshal() ([]byte, error) {
	var (
		err        error
		data, meta []byte
	)

	if e.Data != nil {
		if data, err = e.Data.Marshal(); err != nil {
			return nil, err
		}
	}

	if e.Meta != nil {
		if meta, err = e.Meta.Marshal(); err != nil {
			return nil, err
		}
	}

	if e.ID == "" {
		e.ID = GenId()
	}

	// Add event time if not set.
	if e.Time.IsZero() {
		e.Time = time.Now()
	}

	return proto.Marshal(&pb.Event{
		Causes:    e.Causes,
		Aggregate: e.Aggregate,
		Msg: &pb.Message{
			Id:   e.ID,
			Type: e.Type,
			Time: e.Time.UnixNano(),
			Data: data,
			Meta: meta,
		},
	})
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

func (d *Data) Set(v interface{}) {
	d.value = v
	d.bytes = nil
}

// Marshal marshals the data into an internal encoded data structure.
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

// Binary returns Data that encodes itself to bytes.
func Binary(m encoding.BinaryMarshaler) *Data {
	return &Data{
		Encoding: "binary",
		value:    m,
	}
}

// Bytes returns Data that encodes and decodes bytes.
func Bytes(b []byte) *Data {
	return &Data{
		Encoding: "bytes",
		value:    b,
	}
}

// String returns Data that encodes and decodes a string.
func String(s string) *Data {
	return &Data{
		Encoding: "string",
		value:    s,
	}
}

// JSON returns Data that encodes and decodes the JSON-encodable value.
func JSON(v interface{}) *Data {
	return &Data{
		Encoding: "json",
		value:    v,
	}
}

// Proto returns Data that encodes and decodes the proto message.
func Proto(m proto.Message) *Data {
	return &Data{
		Encoding: "proto",
		value:    m,
	}
}
