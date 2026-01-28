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
	groups := buildCommandPaletteGroups(m, registry)

	root := make([]picker.CommandItem, 0, 8)
	root = append(root, quickCommandItems(m, groups.specIndex)...)
	if len(groups.agentLaunchItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Add Agent",
			Desc:     "Launch agent workflows",
			Children: groups.agentLaunchItems,
		})
	}
	if len(groups.paneItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Panes",
			Desc:     "Pane commands",
			Children: groups.paneItems,
		})
	}
	if len(groups.sessionItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Sessions",
			Desc:     "Session commands",
			Children: groups.sessionItems,
		})
	}
	if len(groups.projectItems) > 0 {
		root = append(root, picker.CommandItem{
			Label:    "Project",
			Desc:     "Project commands",
			Children: groups.projectItems,
		})
	}
	root = append(root, groups.menuItems...)
	root = append(root, groups.agentItems...)
	root = append(root, groups.otherItems...)
	if groups.exitItem != nil {
		root = append(root, *groups.exitItem)
	}
	return commandItemsToList(root)
}

type commandPaletteGroups struct {
	paneItems        []picker.CommandItem
	sessionItems     []picker.CommandItem
	projectItems     []picker.CommandItem
	menuItems        []picker.CommandItem
	agentItems       []picker.CommandItem
	agentLaunchItems []picker.CommandItem
	otherItems       []picker.CommandItem
	exitItem         *picker.CommandItem
	specIndex        map[commandID]commandSpec
}

func buildCommandPaletteGroups(m *Model, registry commandRegistry) commandPaletteGroups {
	groups := commandPaletteGroups{
		otherItems: make([]picker.CommandItem, 0, 16),
		specIndex:  make(map[commandID]commandSpec),
	}
	for _, group := range registry.Groups {
		addCommandPaletteGroup(&groups, m, group)
	}
	return groups
}

func addCommandPaletteGroup(groups *commandPaletteGroups, m *Model, group commandGroup) {
	if groups == nil {
		return
	}
	switch group.Name {
	case "pane":
		groups.paneItems = append(groups.paneItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	case "session":
		groups.sessionItems = append(groups.sessionItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	case "project":
		groups.projectItems = append(groups.projectItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	case "menu":
		groups.menuItems = append(groups.menuItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	case "other":
		addOtherCommandPaletteItems(groups, m, group.Commands)
	case "agent":
		groups.agentItems = append(groups.agentItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	case "agent_launch":
		groups.agentLaunchItems = append(groups.agentLaunchItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	default:
		groups.otherItems = append(groups.otherItems, commandSpecsToItems(m, group.Commands)...)
		addSpecIndex(groups.specIndex, group.Commands)
	}
}

func addOtherCommandPaletteItems(groups *commandPaletteGroups, m *Model, commands []commandSpec) {
	if groups == nil {
		return
	}
	for _, cmd := range commands {
		addSpecIndex(groups.specIndex, []commandSpec{cmd})
		if cmd.ID == "other_quit" {
			item := commandItemWithLabel(m, cmd, cmd.Label)
			groups.exitItem = &item
			continue
		}
		groups.otherItems = append(groups.otherItems, commandSpecsToItems(m, []commandSpec{cmd})...)
	}
}

func addSpecIndex(index map[commandID]commandSpec, commands []commandSpec) {
	for _, cmd := range commands {
		index[cmd.ID] = cmd
	}
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
		addSpecIndex(specIndex, group.Commands)
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
