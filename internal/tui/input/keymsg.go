package input

import (
	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
)

// KeyMsg carries the raw ultraviolet key (including modifiers) so the app can
// distinguish Ctrl+Shift chords that Bubble Tea v1 can't represent.
type KeyMsg struct {
	Key   uv.Key
	Paste bool
}

func (m KeyMsg) Tea() tea.KeyMsg {
	t := tea.KeyMsg(toTeaKey(m.Key))
	if m.Paste {
		t.Paste = true
	}
	return t
}

func (m KeyMsg) Keystroke() string {
	return m.Key.Keystroke()
}
