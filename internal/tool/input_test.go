package tool

import (
	"bytes"
	"testing"

	"github.com/regenrek/peakypanes/internal/limits"
)

func TestApplyProfileBracketedPaste(t *testing.T) {
	profile := Profile{BracketedPaste: true, Submit: []byte{'\n'}}
	payload := ApplyProfile([]byte("hi"), profile, false)
	if string(payload) != "\x1b[200~hi\x1b[201~" {
		t.Fatalf("payload = %q", payload)
	}
	raw := ApplyProfile([]byte("hi"), profile, true)
	if string(raw) != "hi" {
		t.Fatalf("raw payload = %q", raw)
	}
}

func TestDetectFromInput(t *testing.T) {
	reg := defaultRegistry(t)
	if got := reg.DetectFromInput("codex"); got != "codex" {
		t.Fatalf("DetectFromInput = %q", got)
	}
	if got := reg.DetectFromInput("codex\nfoo"); got != "" {
		t.Fatalf("DetectFromInput(multiline) = %q", got)
	}
}

func TestDetectFromInputBytes(t *testing.T) {
	reg := defaultRegistry(t)
	if got := reg.DetectFromInputBytes([]byte("codex"), 16); got != "codex" {
		t.Fatalf("DetectFromInputBytes = %q", got)
	}
	if got := reg.DetectFromInputBytes([]byte("codex\nfoo"), 16); got != "" {
		t.Fatalf("DetectFromInputBytes(multiline) = %q", got)
	}
	payload := append(bytes.Repeat([]byte("a"), limits.PayloadInspectLimit), []byte("codex")...)
	if got := reg.DetectFromInputBytes(payload, limits.PayloadInspectLimit); got != "" {
		t.Fatalf("DetectFromInputBytes(limit) = %q", got)
	}
	oversize := append([]byte("codex "), bytes.Repeat([]byte("x"), limits.PayloadInspectLimit)...)
	if got := reg.DetectFromInputBytes(oversize, limits.PayloadInspectLimit); got != "codex" {
		t.Fatalf("DetectFromInputBytes(oversize) = %q", got)
	}
}
