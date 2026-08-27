package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/snappy"
)

// Event describes a durable mutation.
type Event uint8

const (
	EventWrite Event = iota + 1
	EventDelete
)

// Record is one durable mutation.
type Record struct {
	Event   Event
	Key     string
	Payload []byte
}

const headerSize = 4 + 1 + 4

func (r Record) Encode() ([]byte, error) {
	raw := make([]byte, headerSize+len(r.Key)+len(r.Payload))
	binary.BigEndian.PutUint32(raw[0:4], uint32(len(r.Key)))
	raw[4] = byte(r.Event)
	binary.BigEndian.PutUint32(raw[5:9], uint32(len(r.Payload)))
	copy(raw[headerSize:], r.Key)
	copy(raw[headerSize+len(r.Key):], r.Payload)
	return snappy.Encode(nil, raw), nil
}

func Decode(blob []byte) (Record, error) {
	raw, err := snappy.Decode(nil, blob)
	if err != nil {
		return Record{}, fmt.Errorf("decompress wal record: %w", err)
	}
	if len(raw) < headerSize {
		return Record{}, io.ErrUnexpectedEOF
	}
	keyLen := int(binary.BigEndian.Uint32(raw[0:4]))
	payloadLen := int(binary.BigEndian.Uint32(raw[5:9]))
	if headerSize+keyLen+payloadLen != len(raw) {
		return Record{}, fmt.Errorf("wal record length mismatch")
	}
	return Record{
		Event:   Event(raw[4]),
		Key:     string(raw[headerSize : headerSize+keyLen]),
		Payload: bytes.Clone(raw[headerSize+keyLen:]),
	}, nil
}
