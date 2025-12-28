package sessiond

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// EnvelopeKind distinguishes request/response/event frames.
type EnvelopeKind uint8

const (
	EnvelopeRequest EnvelopeKind = iota + 1
	EnvelopeResponse
	EnvelopeEvent
)

// Envelope is the framed message payload exchanged between client and daemon.
type Envelope struct {
	Kind    EnvelopeKind
	Op      Op
	Event   EventType
	ID      uint64
	Payload []byte
	Error   string
}

func encodePayload(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	return buf.Bytes(), nil
}

func decodePayload(data []byte, v any) error {
	if v == nil || len(data) == 0 {
		return nil
	}
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}
