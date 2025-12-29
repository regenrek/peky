package sessiond

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
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

const maxEnvelopeSize = 64 << 20

var errEnvelopeTooLarge = errors.New("sessiond: envelope too large")

func encodeEnvelope(env Envelope) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(env); err != nil {
		return nil, fmt.Errorf("encode envelope: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&env); err != nil {
		return Envelope{}, fmt.Errorf("decode envelope: %w", err)
	}
	return env, nil
}

func writeEnvelope(w io.Writer, env Envelope) error {
	payload, err := encodeEnvelope(env)
	if err != nil {
		return err
	}
	if len(payload) > maxEnvelopeSize {
		return errEnvelopeTooLarge
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeFull(w, header[:]); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	return writeFull(w, payload)
}

func readEnvelope(r io.Reader) (Envelope, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Envelope{}, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > maxEnvelopeSize {
		return Envelope{}, errEnvelopeTooLarge
	}
	if size == 0 {
		return Envelope{}, nil
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Envelope{}, err
	}
	return decodeEnvelope(payload)
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
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
