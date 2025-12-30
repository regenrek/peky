package app

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

// ===== Command palette =====

const commandPaletteHeading = "⌘ Command Palette"

func (m *Model) setupCommandPalette() {
	m.commandPalette = picker.NewCommandPalette()
}

func (m *Model) openCommandPalette() tea.Cmd {
	m.setCommandPaletteSize()
	m.commandPalette.ResetFilter()
	m.commandPalette.SetFilterState(list.Filtering)
	cmd := m.commandPalette.SetItems(m.commandPaletteItems())
	m.setState(StateCommandPalette)
	return cmd
}

func (m *Model) setCommandPaletteSize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	headerHeight := lipgloss.Height(theme.HelpTitle.Render(commandPaletteHeading))
	hFrame, vFrame := dialogStyleCompact.GetFrameSize()
	availableW := m.width - 6
	availableH := m.height - 4
	if availableW < 30 {
		availableW = m.width
	}
	if availableH < 10 {
		availableH = m.height
	}
	desiredW := clamp(availableW, 46, 100)
	desiredH := clamp(availableH, 12, 26)
	listW := desiredW - hFrame
	listH := desiredH - vFrame - headerHeight
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	maxListH := m.height - vFrame - headerHeight
	if maxListH < 1 {
		maxListH = 1
	}
	if listH > maxListH {
		listH = maxListH
	}
	if listH < 6 {
		if maxListH < 6 {
			listH = maxListH
		} else {
			listH = 6
		}
	}
	m.commandPalette.SetSize(listW, listH)
}

func (m *Model) commandPaletteItems() []list.Item {
	registry, err := m.commandRegistry()
	if err != nil {
		m.setToast("Command registry error: "+err.Error(), toastError)
		return nil
	}
	items := make([]list.Item, 0, 16)
	for _, group := range registry.Groups {
		for _, cmd := range group.Commands {
			cmd := cmd
			items = append(items, picker.CommandItem{
				Label:    cmd.Label,
				Desc:     cmd.Desc,
				Shortcut: cmd.Shortcut,
				Run: func() tea.Cmd {
					if cmd.Run == nil {
						return nil
					}
					return cmd.Run(m, commandArgs{})
				},
			})
		}
	}
	return items
}

func commandItemsToList(items []picker.CommandItem) []list.Item {
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func (m *Model) updateCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commandPalette.FilterState() == list.Filtering {
		switch msg.String() {
		case "enter":
			return m, m.runCommandPaletteSelection()
		case "esc":
			m.commandPalette.ResetFilter()
			m.clearSlashPaletteInput(false)
			m.setState(StateDashboard)
			return m, nil
		}
		if handled := m.handleCommandPaletteFilterNavigation(msg); handled {
			return m, nil
		}
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc", "q":
		m.commandPalette.ResetFilter()
		m.clearSlashPaletteInput(false)
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		return m, m.runCommandPaletteSelection()
	}

	var cmd tea.Cmd
	m.commandPalette, cmd = m.commandPalette.Update(msg)
	return m, cmd
}

func (m *Model) runCommandPaletteSelection() tea.Cmd {
	item, ok := m.commandPalette.SelectedItem().(picker.CommandItem)
	m.commandPalette.ResetFilter()
	m.clearSlashPaletteInput(true)
	m.setState(StateDashboard)
	if !ok || item.Run == nil {
		return nil
	}
	return item.Run()
}

func (m *Model) handleCommandPaletteFilterNavigation(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		m.commandPalette.CursorUp()
		return true
	case "down":
		m.commandPalette.CursorDown()
		return true
	case "pgup":
		m.commandPalette.Paginator.PrevPage()
		return true
	case "pgdown":
		m.commandPalette.Paginator.NextPage()
		return true
	case "home":
		m.commandPalette.Select(0)
		return true
	case "end":
		items := m.commandPalette.VisibleItems()
		if len(items) == 0 {
			return true
		}
		m.commandPalette.Select(len(items) - 1)
		return true
	default:
		return false
	}
}
