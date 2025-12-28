package vt

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func TestCsiDeviceReports(t *testing.T) {
	emu := NewEmulator(4, 2)
	out := readAfter(t, emu, func() {
		emu.handleCsi(ansi.Cmd('c'), ansi.Params{ansi.Param(0)})
	})
	if out == "" {
		t.Fatalf("expected primary device attributes output")
	}

	out = readAfter(t, emu, func() {
		emu.handleCsi(ansi.Cmd('n'), ansi.Params{ansi.Param(6)})
	})
	if out == "" {
		t.Fatalf("expected cursor position report")
	}

	out = readAfter(t, emu, func() {
		emu.handleCsi(ansi.Cmd(ansi.Command('?', 0, 'n')), ansi.Params{ansi.Param(6)})
	})
	if out == "" {
		t.Fatalf("expected extended cursor position report")
	}
}

func TestKeySequencesWithModifiers(t *testing.T) {
	key := uv.KeyPressEvent{Code: KeyTab, Mod: ModShift}
	if seq := sequenceForKeyPress(key, false, false); seq != "\x1b[Z" {
		t.Fatalf("expected shift+tab sequence, got %q", seq)
	}
	key = uv.KeyPressEvent{Code: 'x', Mod: ModAlt}
	if seq := sequenceForKeyPress(key, false, false); seq != "\x1bx" {
		t.Fatalf("expected alt prefix sequence, got %q", seq)
	}
	key = uv.KeyPressEvent{Code: 'x', Mod: ModShift}
	if seq := rawSequenceForKeyPress(key, false, false); seq != "" {
		t.Fatalf("expected raw sequence empty for non-handled mod, got %q", seq)
	}
}
