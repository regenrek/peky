package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEncodeKeyMsg_ShiftTab(t *testing.T) {
	got := encodeKeyMsg(tea.KeyMsg{Type: tea.KeyShiftTab})
	if string(got) != "\x1b[Z" {
		t.Fatalf("got=%q", got)
	}
}

func TestEncodeKeyMsg_CtrlUp(t *testing.T) {
	got := encodeKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if string(got) != "\x1b[1;5A" {
		t.Fatalf("got=%q", got)
	}
}

func TestEncodeKeyMsg_AltUp(t *testing.T) {
	got := encodeKeyMsg(tea.KeyMsg{Type: tea.KeyUp, Alt: true})
	if string(got) != "\x1b[1;3A" {
		t.Fatalf("got=%q", got)
	}
}
