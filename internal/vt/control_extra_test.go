package vt

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHandleControlCharacters(t *testing.T) {
	emu := NewEmulator(4, 2)
	bellCalls := 0
	emu.cb.Bell = func() { bellCalls++ }

	emu.handleControl(byte(ansi.BEL))
	if bellCalls != 1 {
		t.Fatalf("expected bell callback")
	}

	emu.scr.setCursor(2, 0, false)
	emu.handleControl(byte(ansi.HTS))
	if !emu.tabstops.IsStop(2) {
		t.Fatalf("expected tab stop set")
	}

	emu.setMode(ansi.ModeLineFeedNewLine, ansi.ModeSet)
	emu.scr.setCursor(3, 0, false)
	emu.handleControl(byte(ansi.LF))
	x, y := emu.scr.CursorPosition()
	if x != 0 || y != 1 {
		t.Fatalf("expected linefeed+carriage return, got %d,%d", x, y)
	}

	emu.scr.setCursor(1, 1, false)
	emu.handleControl(byte(ansi.RI))
	_, y = emu.scr.CursorPosition()
	if y != 0 {
		t.Fatalf("expected reverse index to move cursor up")
	}
	emu.handleControl(byte(ansi.IND))
	emu.handleControl(byte(ansi.SS2))
	if emu.gsingle != 2 {
		t.Fatalf("expected ss2 to set gsingle=2")
	}
	emu.handleControl(byte(ansi.SS3))
	if emu.gsingle != 3 {
		t.Fatalf("expected ss3 to set gsingle=3")
	}
}

func TestHandleEscSequences(t *testing.T) {
	emu := NewEmulator(2, 1)

	emu.handleEsc(ansi.Cmd('n'))
	if emu.gl != 2 {
		t.Fatalf("expected LS2 to set gl=2, got %d", emu.gl)
	}
	emu.handleEsc(ansi.Cmd('o'))
	if emu.gl != 3 {
		t.Fatalf("expected LS3 to set gl=3, got %d", emu.gl)
	}
	emu.handleEsc(ansi.Cmd('|'))
	if emu.gr != 3 {
		t.Fatalf("expected LS3R to set gr=3, got %d", emu.gr)
	}
	emu.handleEsc(ansi.Cmd('~'))
	if emu.gr != 1 {
		t.Fatalf("expected LS1R to set gr=1, got %d", emu.gr)
	}
}

func TestHandleModeSkipsPermanentAndMissing(t *testing.T) {
	emu := NewEmulator(2, 1)
	emu.modes[ansi.ModeTextCursorEnable] = ansi.ModePermanentlySet
	emu.handleMode(ansi.Params{ansi.Param(int(ansi.ModeTextCursorEnable))}, true, false)
	if emu.modes[ansi.ModeTextCursorEnable] != ansi.ModePermanentlySet {
		t.Fatalf("expected permanent mode unchanged")
	}

	emu.handleMode(ansi.Params{ansi.Param(-1)}, true, false)
}

func TestRepeatPreviousCharacterNoLast(t *testing.T) {
	emu := NewEmulator(2, 1)
	emu.repeatPreviousCharacter(2)
	x, y := emu.scr.CursorPosition()
	if x != 0 || y != 0 {
		t.Fatalf("expected cursor unchanged")
	}
}
