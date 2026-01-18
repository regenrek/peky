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

type commandPaletteState struct {
	items []list.Item
}

func (m *Model) setupCommandPalette() {
	m.commandPalette = picker.NewCommandPalette()
}

func (m *Model) openCommandPalette() tea.Cmd {
	m.setCommandPaletteSize()
	m.commandPalette.ResetFilter()
	m.commandPalette.SetFilterState(list.Filtering)
	m.commandPaletteStack = nil
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
	var paneItems []picker.CommandItem
	var sessionItems []picker.CommandItem
	var projectItems []picker.CommandItem
	otherItems := make([]picker.CommandItem, 0, 16)

	for _, group := range registry.Groups {
		switch group.Name {
		case "pane":
			paneItems = append(paneItems, commandSpecsToItems(m, group.Commands)...)
		case "session":
			sessionItems = append(sessionItems, commandSpecsToItems(m, group.Commands)...)
		case "project":
			projectItems = append(projectItems, commandSpecsToItems(m, group.Commands)...)
		default:
			otherItems = append(otherItems, commandSpecsToItems(m, group.Commands)...)
		}
	}

	root := make([]picker.CommandItem, 0, 8)
	if len(paneItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Pane",
			Desc:     "Pane commands",
			Children: paneItems,
		})
	}
	if len(sessionItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Session",
			Desc:     "Session commands",
			Children: sessionItems,
		})
	}
	if len(projectItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Project",
			Desc:     "Project commands",
			Children: projectItems,
		})
	}
	root = append(root, otherItems...)
	return commandItemsToList(root)
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
			if m.commandPaletteHasParent() {
				return m, m.closeCommandPaletteCategory()
			}
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
		if m.commandPaletteHasParent() {
			return m, m.closeCommandPaletteCategory()
		}
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
	if !ok || item.Run == nil {
		m.commandPalette.ResetFilter()
		m.setState(StateDashboard)
		return nil
	}
	if item.Back {
		return m.closeCommandPaletteCategory()
	}
	if len(item.Children) > 0 {
		return m.openCommandPaletteCategory(item)
	}
	m.commandPalette.ResetFilter()
	m.setState(StateDashboard)
	return item.Run()
}

func commandSpecsToItems(m *Model, specs []commandSpec) []picker.CommandItem {
	items := make([]picker.CommandItem, 0, len(specs))
	for _, cmd := range specs {
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
	return items
}

func (m *Model) commandPaletteHasParent() bool {
	return m != nil && len(m.commandPaletteStack) > 0
}

func (m *Model) openCommandPaletteCategory(item picker.CommandItem) tea.Cmd {
	if m == nil || len(item.Children) == 0 {
		return nil
	}
	m.commandPaletteStack = append(m.commandPaletteStack, commandPaletteState{items: m.commandPalette.Items()})
	children := append([]picker.CommandItem{{
		Label: "← Back",
		Desc:  "Command categories",
		Back:  true,
	}}, item.Children...)
	m.commandPalette.ResetFilter()
	m.commandPalette.SetFilterState(list.Filtering)
	return m.commandPalette.SetItems(commandItemsToList(children))
}

func (m *Model) closeCommandPaletteCategory() tea.Cmd {
	if m == nil || len(m.commandPaletteStack) == 0 {
		return nil
	}
	last := m.commandPaletteStack[len(m.commandPaletteStack)-1]
	m.commandPaletteStack = m.commandPaletteStack[:len(m.commandPaletteStack)-1]
	m.commandPalette.ResetFilter()
	m.commandPalette.SetFilterState(list.Filtering)
	return m.commandPalette.SetItems(last.items)
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
