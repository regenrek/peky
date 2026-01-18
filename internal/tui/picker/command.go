package picker

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandItem represents a selectable command in the palette.
type CommandItem struct {
	Label     string
	Desc      string
	Shortcut  string
	HelpKey   string
	HelpValue string
	Run       func() tea.Cmd
	Children  []CommandItem
	Back      bool
}

func (c CommandItem) Title() string       { return c.Label }
func (c CommandItem) Description() string { return c.Desc }
func (c CommandItem) FilterValue() string {
	base := strings.ToLower(strings.TrimSpace(c.Label + " " + c.Desc + " " + c.Shortcut))
	if len(c.Children) == 0 {
		return base
	}
	parts := make([]string, 0, 1+len(c.Children))
	if base != "" {
		parts = append(parts, base)
	}
	for _, child := range c.Children {
		if value := strings.TrimSpace(child.FilterValue()); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}
