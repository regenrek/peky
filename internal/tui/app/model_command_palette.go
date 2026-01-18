package app

import (
	"strings"

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
	m.commandPaletteFlat = false
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
	var menuItems []picker.CommandItem
	otherItems := make([]picker.CommandItem, 0, 16)
	var agentItems []picker.CommandItem
	var exitItem *picker.CommandItem
	specIndex := make(map[commandID]commandSpec)

	for _, group := range registry.Groups {
		switch group.Name {
		case "pane":
			paneItems = append(paneItems, commandSpecsToItems(m, group.Commands)...)
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
			}
		case "session":
			sessionItems = append(sessionItems, commandSpecsToItems(m, group.Commands)...)
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
			}
		case "project":
			projectItems = append(projectItems, commandSpecsToItems(m, group.Commands)...)
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
			}
		case "menu":
			menuItems = append(menuItems, commandSpecsToItems(m, group.Commands)...)
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
			}
		case "other":
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
				if cmd.ID == "other_quit" {
					item := commandItemWithLabel(m, cmd, cmd.Label)
					exitItem = &item
					continue
				}
				otherItems = append(otherItems, commandSpecsToItems(m, []commandSpec{cmd})...)
			}
		case "agent":
			agentItems = append(agentItems, commandSpecsToItems(m, group.Commands)...)
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
			}
		default:
			otherItems = append(otherItems, commandSpecsToItems(m, group.Commands)...)
			for _, cmd := range group.Commands {
				specIndex[cmd.ID] = cmd
			}
		}
	}

	root := make([]picker.CommandItem, 0, 8)
	root = append(root, quickCommandItems(m, specIndex)...)
	if len(paneItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Panes",
			Desc:     "Pane commands",
			Children: paneItems,
		})
	}
	if len(sessionItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Sessions",
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
	root = append(root, menuItems...)
	root = append(root, agentItems...)
	root = append(root, otherItems...)
	if exitItem != nil {
		root = append(root, *exitItem)
	}
	return commandItemsToList(root)
}

func (m *Model) commandPaletteFlatItems() []list.Item {
	registry, err := m.commandRegistry()
	if err != nil {
		m.setToast("Command registry error: "+err.Error(), toastError)
		return nil
	}
	items := make([]picker.CommandItem, 0, 32)
	specIndex := make(map[commandID]commandSpec)
	for _, group := range registry.Groups {
		for _, cmd := range group.Commands {
			specIndex[cmd.ID] = cmd
		}
		items = append(items, commandSpecsToItems(m, group.Commands)...)
	}
	items = append(quickCommandItems(m, specIndex), items...)
	return commandItemsToList(items)
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
				cmd := m.closeCommandPaletteCategory()
				return m, tea.Batch(cmd, m.ensureCommandPaletteMode())
			}
			m.setState(StateDashboard)
			return m, nil
		}
		if handled := m.handleCommandPaletteFilterNavigation(msg); handled {
			return m, nil
		}
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
		return m, tea.Batch(cmd, m.ensureCommandPaletteMode())
	}

	switch msg.String() {
	case "esc", "q":
		m.commandPalette.ResetFilter()
		if m.commandPaletteHasParent() {
			cmd := m.closeCommandPaletteCategory()
			return m, tea.Batch(cmd, m.ensureCommandPaletteMode())
		}
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		return m, m.runCommandPaletteSelection()
	}

	var cmd tea.Cmd
	m.commandPalette, cmd = m.commandPalette.Update(msg)
	return m, tea.Batch(cmd, m.ensureCommandPaletteMode())
}

func (m *Model) runCommandPaletteSelection() tea.Cmd {
	item, ok := m.commandPalette.SelectedItem().(picker.CommandItem)
	if item.Back {
		return m.closeCommandPaletteCategory()
	}
	if len(item.Children) > 0 {
		return m.openCommandPaletteCategory(item)
	}
	if !ok || item.Run == nil {
		m.commandPalette.ResetFilter()
		m.setState(StateDashboard)
		return nil
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
	m.commandPaletteFlat = false
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

func (m *Model) ensureCommandPaletteMode() tea.Cmd {
	if m == nil || m.commandPaletteHasParent() {
		return nil
	}
	query := strings.TrimSpace(m.commandPalette.FilterValue())
	if query == "" && m.commandPaletteFlat {
		m.commandPaletteFlat = false
		m.commandPalette.ResetFilter()
		m.commandPalette.SetFilterState(list.Filtering)
		return m.commandPalette.SetItems(m.commandPaletteItems())
	}
	if query != "" && !m.commandPaletteFlat {
		m.commandPaletteFlat = true
		cmd := m.commandPalette.SetItems(m.commandPaletteFlatItems())
		m.commandPalette.SetFilterText(query)
		m.commandPalette.SetFilterState(list.Filtering)
		return cmd
	}
	return nil
}

func quickCommandItems(m *Model, specs map[commandID]commandSpec) []picker.CommandItem {
	quick := make([]picker.CommandItem, 0, 3)
	if cmd, ok := specs["pane_add"]; ok {
		quick = append(quick, commandItemWithLabel(m, cmd, "Add Pane"))
	}
	if cmd, ok := specs["pane_close"]; ok {
		quick = append(quick, commandItemWithLabel(m, cmd, "Close Pane"))
	}
	if cmd, ok := specs["session_new"]; ok {
		quick = append(quick, commandItemWithLabel(m, cmd, "Add Session"))
	}
	return quick
}

func commandItemWithLabel(m *Model, cmd commandSpec, label string) picker.CommandItem {
	item := commandSpecsToItems(m, []commandSpec{cmd})
	if len(item) == 0 {
		return picker.CommandItem{}
	}
	item[0].Label = label
	return item[0]
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
