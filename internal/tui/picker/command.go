package picker

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandItem represents a selectable command in the palette.
type CommandItem struct {
	Label string
	Desc  string
	Run   func() tea.Cmd
}

func (c CommandItem) Title() string       { return c.Label }
func (c CommandItem) Description() string { return c.Desc }
func (c CommandItem) FilterValue() string { return strings.ToLower(c.Label + " " + c.Desc) }
