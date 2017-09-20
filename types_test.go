package eda

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func testMessage(t *testing.T, m *Message) {
	b, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	var m2 Message
	if err := m2.Unmarshal(b); err != nil {
		t.Fatal(err)
	}

	tests := []func(t *testing.T){
		func(t *testing.T) {
			if m.ID != m2.ID {
				t.Errorf("expected id %s, got %s", m.ID, m2.ID)
			}
		},
		func(t *testing.T) {
			if m.Type != m2.Type {
				t.Errorf("expected type %s, got %s", m.Type, m2.Type)
			}
		},
		func(t *testing.T) {
			if !m.Time.Equal(m2.Time) {
				t.Errorf("expected time %s, got %s", m.Time, m2.Time)
			}
		},
	}

	for i, fn := range tests {
		t.Run(fmt.Sprint(i), fn)
	}
}

func TestMessage(t *testing.T) {
	tests := []*Message{
		&Message{
			Type: "test",
		},
		&Message{
			ID:   "123",
			Type: "test",
		},
		&Message{
			Type: "test",
			Time: time.Now(),
		},
		&Message{
			Type: "test",
			Data: String("some data"),
		},
	}

	for _, m := range tests {
		testMessage(t, m)
	}
}

func compareData(t *testing.T, d1, d2 *Data) {
	b1, err := d1.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	b2, err := d2.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Errorf("data does not match")
	}
}

func testData(t *testing.T, d *Data, v interface{}) {
	b, err := d.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	var d2 Data
	if err := d2.Unmarshal(b); err != nil {
		t.Fatal(err)
	}

	if err := d2.Decode(v); err != nil {
		t.Fatal(err)
	}
}

func TestString(t *testing.T) {
	d := &Data{
		Encoding: "string",
		value:    "foobar",
	}

	var v string
	testData(t, d, &v)
	if !reflect.DeepEqual(d.value, v) {
		t.Fatal("values don't match")
	}
}

func TestBytes(t *testing.T) {
	d := &Data{
		Encoding: "bytes",
		value:    []byte{0x1, 0x2, 0x3},
	}

	var v []byte
	testData(t, d, &v)
	if !reflect.DeepEqual(d.value, v) {
		t.Fatal("values don't match")
	}
}

func TestJSON(t *testing.T) {
	d := &Data{
		Encoding: "json",
		value: map[string]int{
			"foo": 1,
			"bar": 2,
		},
	}

	var v map[string]int
	testData(t, d, &v)
	if !reflect.DeepEqual(d.value, v) {
		t.Fatal("values don't match")
	}
}
