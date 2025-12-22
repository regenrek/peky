package peakypanes

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandPaletteDelegateRender(t *testing.T) {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = true

	items := []list.Item{
		CommandItem{Label: "Test", Desc: "First line\nSecond line", Run: func(*Model) tea.Cmd { return nil }},
	}

	model := list.New(items, delegate, 20, 4)
	model.SetSize(20, 4)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])
	if buf.Len() == 0 {
		t.Fatalf("Render() produced empty output")
	}
}
