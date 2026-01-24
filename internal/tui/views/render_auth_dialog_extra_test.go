package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestViewAuthDialogDefaults(t *testing.T) {
	input := textinput.New()
	input.SetValue("code")
	m := Model{
		Width:      80,
		Height:     24,
		AuthDialog: AuthDialog{Input: input},
	}
	out := m.viewAuthDialog()
	if !strings.Contains(out, "OAuth") {
		t.Fatalf("expected default auth dialog title")
	}
	if !strings.Contains(out, "Paste code") {
		t.Fatalf("expected auth dialog prompt")
	}
}
