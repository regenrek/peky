package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestNormalizeKeyString(t *testing.T) {
	if _, err := normalizeKeyString(" "); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if got, err := normalizeKeyString("space"); err != nil || got != " " {
		t.Fatalf("expected space normalized, got %q err=%v", got, err)
	}
	if got, err := normalizeKeyString("alt+K"); err != nil || got != "alt+K" {
		t.Fatalf("expected alt+K preserved, got %q err=%v", got, err)
	}
	if got, err := normalizeKeyString("Enter"); err != nil || got != "enter" {
		t.Fatalf("expected enter normalized, got %q err=%v", got, err)
	}
	if _, err := normalizeKeyString("alt+"); err == nil {
		t.Fatalf("expected error for invalid alt+ key")
	}
}

func TestKeyLabelHelpers(t *testing.T) {
	if !isSingleRune("k") || isSingleRune("kk") {
		t.Fatalf("unexpected isSingleRune result")
	}
	if prettyKeyLabel("shift+tab") != "⇧tab" {
		t.Fatalf("expected pretty shift+tab")
	}
	if prettyKeyLabel(" ") != "space" {
		t.Fatalf("expected pretty space")
	}

	binding := key.NewBinding(key.WithHelp("x", "desc"))
	if got := keyLabel(binding); got != "x" {
		t.Fatalf("expected help label, got %q", got)
	}

	binding = key.NewBinding(key.WithKeys("a", "b"))
	if got := keyLabel(binding); got == "" {
		t.Fatalf("expected key label from keys")
	}
}
