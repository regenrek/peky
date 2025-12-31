package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const SchemaVersion = "1.0.0"

type Meta struct {
	Command       string    `json:"command"`
	SchemaVersion string    `json:"schema_version"`
	Version       string    `json:"version,omitempty"`
	RequestID     string    `json:"request_id,omitempty"`
	DurationMS    float64   `json:"duration_ms,omitempty"`
	TS            time.Time `json:"ts"`
	Stream        bool      `json:"stream,omitempty"`
	Seq           int64     `json:"seq,omitempty"`
	EOF           bool      `json:"eof,omitempty"`
}

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Ok    bool      `json:"ok"`
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

type SuccessEnvelope struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

func NewMeta(command, version string) Meta {
	return Meta{
		Command:       command,
		SchemaVersion: SchemaVersion,
		Version:       version,
		TS:            time.Now().UTC(),
	}
}

func NewStreamMeta(command, version string, seq int64, eof bool) Meta {
	meta := NewMeta(command, version)
	meta.Stream = true
	meta.Seq = seq
	meta.EOF = eof
	return meta
}

func WithDuration(meta Meta, start time.Time) Meta {
	meta.DurationMS = float64(time.Since(start).Milliseconds())
	return meta
}

func WriteSuccess(w io.Writer, meta Meta, data any) error {
	return writeJSON(w, SuccessEnvelope{Ok: true, Data: data, Meta: meta})
}

func WriteError(w io.Writer, meta Meta, code, message string, details map[string]any) error {
	if code == "" {
		code = "unknown"
	}
	if message == "" {
		message = "unknown error"
	}
	return writeJSON(w, ErrorEnvelope{
		Ok: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: meta,
	})
}

func writeJSON(w io.Writer, payload any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
