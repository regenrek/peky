package picker

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestCommandPaletteDelegateRender(t *testing.T) {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = true

	items := []list.Item{
		CommandItem{Label: "Test", Desc: "First line\nSecond line", Run: func() tea.Cmd { return nil }},
	}

	model := list.New(items, delegate, 20, 4)
	model.SetSize(20, 4)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])
	if buf.Len() == 0 {
		t.Fatalf("Render() produced empty output")
	}
}

func TestCommandPaletteDelegateRenderTitleOnly(t *testing.T) {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = false

	items := []list.Item{
		CommandItem{Label: "TitleOnly", Desc: "Hidden", Run: func() tea.Cmd { return nil }},
	}

	model := list.New(items, delegate, 20, 2)
	model.SetSize(20, 2)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])
	if buf.Len() == 0 {
		t.Fatalf("Render() produced empty output")
	}
}

func TestCommandPaletteDelegateRenderNoWidth(t *testing.T) {
	delegate := newCommandPaletteDelegate()

	items := []list.Item{
		CommandItem{Label: "Hidden", Desc: "Hidden", Run: func() tea.Cmd { return nil }},
	}

	model := list.New(items, delegate, 0, 1)
	model.SetSize(0, 1)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])
	if buf.Len() != 0 {
		t.Fatalf("Render() should be empty for zero width")
	}
}

func TestCommandPaletteDelegateRenderShortcut(t *testing.T) {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = false

	items := []list.Item{
		CommandItem{Label: "Run", Shortcut: "ctrl+n"},
	}

	model := list.New(items, delegate, 30, 2)
	model.SetSize(30, 2)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])
	if !strings.Contains(buf.String(), "ctrl+n") {
		t.Fatalf("expected shortcut in render output, got %q", buf.String())
	}
}

func TestCommandPaletteDelegateRenderSelectedWidth(t *testing.T) {
	delegate := newCommandPaletteDelegate()
	delegate.ShowDescription = false

	items := []list.Item{
		CommandItem{Label: "Session: New session", Shortcut: "ctrl+n"},
	}

	model := list.New(items, delegate, 30, 2)
	model.SetSize(30, 2)
	model.Select(0)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])

	line := strings.Split(buf.String(), "\n")[0]
	stripped := ansi.Strip(line)
	if got := lipgloss.Width(stripped); got != 30 {
		t.Fatalf("expected rendered width 30, got %d (%q)", got, stripped)
	}
}
