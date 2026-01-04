package app

import (
	"bytes"
	"testing"
)

func TestSetOSCWriterEmitsSequence(t *testing.T) {
	var m Model
	var buf bytes.Buffer

	m.SetOSCWriter(&buf)
	cmd := m.emitOSC("\x1b]22;text\x07")
	if cmd != nil {
		t.Fatalf("expected nil cmd when osc writer is set")
	}
	if got := buf.String(); got != "\x1b]22;text\x07" {
		t.Fatalf("osc output=%q want %q", got, "\x1b]22;text\x07")
	}
}
