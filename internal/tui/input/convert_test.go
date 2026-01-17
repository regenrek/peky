package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
)

func TestToTeaKeySpace(t *testing.T) {
	k := toTeaKey(uv.Key{Code: uv.KeySpace})
	if k.Type != tea.KeySpace {
		t.Fatalf("Type=%v", k.Type)
	}
}

func TestToTeaKeyCtrlU(t *testing.T) {
	k := toTeaKey(uv.Key{Code: 'u', Mod: uv.ModCtrl})
	if k.Type != tea.KeyCtrlU {
		t.Fatalf("Type=%v", k.Type)
	}
}

func TestToTeaKeyShiftTab(t *testing.T) {
	k := toTeaKey(uv.Key{Code: uv.KeyTab, Mod: uv.ModShift})
	if k.Type != tea.KeyShiftTab {
		t.Fatalf("Type=%v", k.Type)
	}
}

func TestToTeaKeyCtrlShiftUp(t *testing.T) {
	k := toTeaKey(uv.Key{Code: uv.KeyUp, Mod: uv.ModCtrl | uv.ModShift})
	if k.Type != tea.KeyCtrlShiftUp {
		t.Fatalf("Type=%v", k.Type)
	}
}

func TestPasteEvent(t *testing.T) {
	msg, ok := toTeaMsg(uv.PasteEvent{Content: "hi\n"})
	if !ok {
		t.Fatalf("expected ok")
	}
	km, ok := msg.(KeyMsg)
	if !ok {
		t.Fatalf("expected KeyMsg, got %T", msg)
	}
	teaMsg := km.Tea()
	if teaMsg.Type != tea.KeyRunes || !teaMsg.Paste || string(teaMsg.Runes) != "hi\n" {
		t.Fatalf("msg=%+v", teaMsg)
	}
}
