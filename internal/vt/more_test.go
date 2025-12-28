package vt

import (
	"image/color"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestKeySequenceHelpers(t *testing.T) {
	if seq, ok := ctrlKeySequence('a'); !ok || seq != "\x01" {
		t.Fatalf("ctrlKeySequence = %q ok=%v", seq, ok)
	}
	if seq, ok := arrowKeySequence(KeyUp, true); !ok || seq == "" {
		t.Fatalf("arrowKeySequence app = %q ok=%v", seq, ok)
	}
	if seq, ok := navKeySequence(KeyHome); !ok || seq == "" {
		t.Fatalf("navKeySequence = %q ok=%v", seq, ok)
	}
	if seq, ok := functionKeySequence(KeyF1); !ok || seq == "" {
		t.Fatalf("functionKeySequence = %q ok=%v", seq, ok)
	}
	if seq, ok := keypadSequence(KeyKp1, true); !ok || seq == "" {
		t.Fatalf("keypadSequence = %q ok=%v", seq, ok)
	}
}

func TestEmulatorControlSequences(t *testing.T) {
	emu := NewEmulator(5, 2)
	emu.scr.setCursor(2, 0, false)
	emu.horizontalTabSet()
	if !emu.tabstops.IsStop(2) {
		t.Fatalf("expected tab stop set")
	}
	emu.backspace()
	if x, _ := emu.scr.CursorPosition(); x != 1 {
		t.Fatalf("backspace cursor = %d", x)
	}

	emu.scr.setCursor(0, 0, false)
	emu.lastChar = 'x'
	emu.repeatPreviousCharacter(2)
	if cell := emu.CellAt(0, 0); cell == nil || cell.Content != "x" {
		t.Fatalf("repeatPreviousCharacter missing cell")
	}
	if cell := emu.CellAt(1, 0); cell == nil || cell.Content != "x" {
		t.Fatalf("repeatPreviousCharacter second cell missing")
	}

	params := ansi.ToParams([]int{int(ansi.ModeCursorKeys)})
	data := readAfter(t, emu, func() {
		emu.handleRequestMode(params, false)
	})
	if data == "" {
		t.Fatalf("expected request mode response")
	}

	emu.setMode(ansi.ModeFocusEvent, ansi.ModeSet)
	data = readAfter(t, emu, func() { emu.Focus() })
	if data == "" {
		t.Fatalf("Focus response empty")
	}
	data = readAfter(t, emu, func() { emu.Blur() })
	if data == "" {
		t.Fatalf("Blur response empty")
	}
}

func TestEmulatorHandlersAndOsc(t *testing.T) {
	emu := NewEmulator(4, 2)
	called := map[string]bool{}

	emu.RegisterDcsHandler(1, func(ansi.Params, []byte) bool {
		called["dcs"] = true
		return true
	})
	emu.RegisterApcHandler(func([]byte) bool {
		called["apc"] = true
		return true
	})
	emu.RegisterSosHandler(func([]byte) bool {
		called["sos"] = true
		return true
	})
	emu.RegisterPmHandler(func([]byte) bool {
		called["pm"] = true
		return true
	})
	emu.RegisterOscHandler(2, func([]byte) bool {
		called["osc"] = true
		return true
	})

	emu.handleDcs(ansi.Cmd(1), ansi.Params{}, []byte("x"))
	emu.handleApc([]byte("a"))
	emu.handleSos([]byte("b"))
	emu.handlePm([]byte("c"))
	emu.handleOsc(2, []byte("2;title"))

	for _, key := range []string{"dcs", "apc", "sos", "pm", "osc"} {
		if !called[key] {
			t.Fatalf("expected handler %s called", key)
		}
	}

	titleCalled := false
	emu.cb.Title = func(string) { titleCalled = true }
	emu.handleTitle(2, []byte("2;mytitle"))
	if !titleCalled || emu.title != "mytitle" {
		t.Fatalf("handleTitle failed")
	}

	emu.handleDefaultColor(10, []byte("10;#010203"))
	if emu.ForegroundColor() == nil {
		t.Fatalf("expected foreground color set")
	}

	emu.SetForegroundColor(color.RGBA{R: 1, G: 2, B: 3, A: 255})
	data := readAfter(t, emu, func() {
		emu.handleDefaultColor(10, []byte("10;?"))
	})
	if data == "" {
		t.Fatalf("expected default color query response")
	}

	wdCalled := false
	emu.cb.WorkingDirectory = func(path string) {
		if path == "/tmp" {
			wdCalled = true
		}
	}
	emu.handleWorkingDirectory(7, []byte("7;/tmp"))
	if !wdCalled || emu.cwd != "/tmp" {
		t.Fatalf("handleWorkingDirectory failed")
	}

	emu.handleHyperlink(8, []byte("8;https://example.com;ref"))
	if emu.scr.cur.Link.URL != "https://example.com" {
		t.Fatalf("handleHyperlink failed")
	}

	emu.lastChar = 'x'
	emu.fullReset()
	if emu.lastChar != 0 || len(emu.grapheme) != 0 {
		t.Fatalf("fullReset did not clear state")
	}
}

func TestDefaultColorHelpers(t *testing.T) {
	if _, ok := defaultColorKindForCmd(999); ok {
		t.Fatalf("expected unknown cmd")
	}
	emu := NewEmulator(2, 1)
	emu.applyDefaultColor(defaultColorBackground, color.Black)
	if emu.BackgroundColor() == nil {
		t.Fatalf("applyDefaultColor failed")
	}
	emu.applyDefaultColor(defaultColorCursor, color.White)
	if emu.CursorColor() == nil {
		t.Fatalf("applyDefaultColor cursor failed")
	}
}

func TestHandleWorkingDirectoryInvalid(t *testing.T) {
	emu := NewEmulator(2, 1)
	emu.handleWorkingDirectory(99, []byte("99;/tmp"))
	if emu.cwd != "" {
		t.Fatalf("unexpected cwd change")
	}
}

func TestHandleHyperlinkInvalid(t *testing.T) {
	emu := NewEmulator(2, 1)
	emu.handleHyperlink(7, []byte("bad"))
	if emu.scr.cur.Link.URL != "" {
		t.Fatalf("unexpected hyperlink set")
	}
}
