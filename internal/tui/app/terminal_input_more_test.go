package app

import "testing"

func TestCtrlSequence(t *testing.T) {
	if got := ctrlSequence("ctrl+a"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("ctrl+a = %#v", got)
	}
	if got := ctrlSequence("ctrl+A"); got != nil {
		t.Fatalf("expected nil for uppercase: %#v", got)
	}
	if got := ctrlSequence("ctrl+ab"); got != nil {
		t.Fatalf("expected nil for invalid length: %#v", got)
	}
}

func TestAltSequence(t *testing.T) {
	if got := altSequence("alt+a"); string(got) != "\x1ba" {
		t.Fatalf("alt+a = %q", got)
	}
	if got := altSequence("alt+ab"); got != nil {
		t.Fatalf("expected nil for invalid length: %#v", got)
	}
}
