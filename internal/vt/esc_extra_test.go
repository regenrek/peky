package vt

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHandleEscSequencesExtra(t *testing.T) {
	emu := NewEmulator(2, 1)
	emu.lastChar = 'x'
	emu.handleEsc(ansi.Cmd('c'))
	if emu.lastChar != 0 {
		t.Fatalf("expected full reset via ESC c")
	}

	emu.handleEsc(ansi.Cmd(ansi.Command(0, '(', 'A')))
	if emu.charsets[0] == nil || emu.charsets[0]['$'] != "£" {
		t.Fatalf("expected UK charset selected")
	}

	// Unhandled ESC should not panic.
	emu.handleEsc(ansi.Cmd('z'))
}
