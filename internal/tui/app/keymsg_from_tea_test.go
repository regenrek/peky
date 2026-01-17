package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
)

func TestKeyMsgFromTea_FunctionKeys(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.KeyMsg
		want rune
	}{
		{name: "f1", msg: tea.KeyMsg{Type: tea.KeyF1}, want: uv.KeyF1},
		{name: "f2", msg: tea.KeyMsg{Type: tea.KeyF2}, want: uv.KeyF2},
		{name: "f3", msg: tea.KeyMsg{Type: tea.KeyF3}, want: uv.KeyF3},
		{name: "f4", msg: tea.KeyMsg{Type: tea.KeyF4}, want: uv.KeyF4},
		{name: "f5", msg: tea.KeyMsg{Type: tea.KeyF5}, want: uv.KeyF5},
		{name: "f6", msg: tea.KeyMsg{Type: tea.KeyF6}, want: uv.KeyF6},
		{name: "f7", msg: tea.KeyMsg{Type: tea.KeyF7}, want: uv.KeyF7},
		{name: "f8", msg: tea.KeyMsg{Type: tea.KeyF8}, want: uv.KeyF8},
		{name: "f9", msg: tea.KeyMsg{Type: tea.KeyF9}, want: uv.KeyF9},
		{name: "f10", msg: tea.KeyMsg{Type: tea.KeyF10}, want: uv.KeyF10},
		{name: "f11", msg: tea.KeyMsg{Type: tea.KeyF11}, want: uv.KeyF11},
		{name: "f12", msg: tea.KeyMsg{Type: tea.KeyF12}, want: uv.KeyF12},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := keyMsgFromTea(tc.msg)
			if got.Key.Code != tc.want {
				t.Fatalf("Code=%v, want %v", got.Key.Code, tc.want)
			}
		})
	}
}

func TestKeyMsgFromTea_AltModifier(t *testing.T) {
	got := keyMsgFromTea(tea.KeyMsg{Type: tea.KeyF2, Alt: true})
	if got.Key.Code != uv.KeyF2 {
		t.Fatalf("Code=%v, want %v", got.Key.Code, uv.KeyF2)
	}
	if !got.Key.Mod.Contains(uv.ModAlt) {
		t.Fatalf("expected alt modifier")
	}
}

func TestKeyMsgFromTea_Runeflow(t *testing.T) {
	got := keyMsgFromTea(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	if got.Key.Code != 'A' {
		t.Fatalf("Code=%v, want %v", got.Key.Code, 'A')
	}
	if got.Key.Text != "A" {
		t.Fatalf("Text=%q, want %q", got.Key.Text, "A")
	}

	extended := keyMsgFromTea(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab")})
	if extended.Key.Code != uv.KeyExtended {
		t.Fatalf("Code=%v, want %v", extended.Key.Code, uv.KeyExtended)
	}
	if extended.Key.Text != "ab" {
		t.Fatalf("Text=%q, want %q", extended.Key.Text, "ab")
	}
}

func TestKeyMsgFromTea_CtrlFallback(t *testing.T) {
	got := keyMsgFromTea(tea.KeyMsg{Type: tea.KeyCtrlA})
	if got.Key.Code != 'a' {
		t.Fatalf("Code=%v, want %v", got.Key.Code, 'a')
	}
	if !got.Key.Mod.Contains(uv.ModCtrl) {
		t.Fatalf("expected ctrl modifier")
	}
}
