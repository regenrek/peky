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
	shortcuts := m.commandPaletteShortcuts()
	items := append([]picker.CommandItem{}, m.paneCommandItems(shortcuts)...)
	items = append(items, m.sessionCommandItems(shortcuts)...)
	items = append(items, m.projectCommandItems(shortcuts)...)
	items = append(items, m.menuCommandItems()...)
	items = append(items, m.otherCommandItems(shortcuts)...)
	return commandItemsToList(items)
}

type commandShortcuts struct {
	openProject    string
	closeProject   string
	newSession     string
	killSession    string
	togglePanes    string
	refresh        string
	editConfig     string
	filterSessions string
	help           string
	quit           string
}

func (m *Model) commandPaletteShortcuts() commandShortcuts {
	shortcuts := commandShortcuts{}
	if m.keys == nil {
		return shortcuts
	}
	shortcuts.openProject = keyLabel(m.keys.openProject)
	shortcuts.closeProject = keyLabel(m.keys.closeProject)
	shortcuts.newSession = keyLabel(m.keys.newSession)
	shortcuts.killSession = keyLabel(m.keys.kill)
	shortcuts.togglePanes = keyLabel(m.keys.togglePanes)
	shortcuts.refresh = keyLabel(m.keys.refresh)
	shortcuts.editConfig = keyLabel(m.keys.editConfig)
	shortcuts.filterSessions = keyLabel(m.keys.filter)
	shortcuts.help = keyLabel(m.keys.help)
	shortcuts.quit = keyLabel(m.keys.quit)
	return shortcuts
}

func (m *Model) paneCommandItems(shortcuts commandShortcuts) []picker.CommandItem {
	return []picker.CommandItem{
		{Label: "Pane: Add pane", Desc: "Add a pane to the current session", Run: func() tea.Cmd {
			return m.addPaneAuto()
		}},
		{Label: "Pane: Swap pane", Desc: "Swap the selected pane with another", Run: func() tea.Cmd {
			m.openPaneSwapPicker()
			return nil
		}},
		{Label: "Pane: Close pane", Desc: "Close the selected pane", Run: func() tea.Cmd {
			return m.openClosePaneConfirm()
		}},
		{Label: "Pane: Rename pane", Desc: "Rename the selected pane title", Run: func() tea.Cmd {
			m.openRenamePane()
			return nil
		}},
		// NOTE: Toggle pane list functionality exists via shortcuts.togglePanes but hidden from menu for now
	}
}

func (m *Model) sessionCommandItems(shortcuts commandShortcuts) []picker.CommandItem {
	return []picker.CommandItem{
		{Label: "Session: New session", Desc: "Pick a layout and create a new session", Shortcut: shortcuts.newSession, Run: func() tea.Cmd {
			m.openLayoutPicker()
			return nil
		}},
		{Label: "Session: Kill session", Desc: "Kill the selected session", Shortcut: shortcuts.killSession, Run: func() tea.Cmd {
			m.openKillConfirm()
			return nil
		}},
		{Label: "Session: Rename session", Desc: "Rename the selected session", Run: func() tea.Cmd {
			m.openRenameSession()
			return nil
		}},
		{Label: "Session: Filter", Desc: "Filter session list", Shortcut: shortcuts.filterSessions, Run: func() tea.Cmd {
			m.filterActive = true
			m.filterInput.Focus()
			m.quickReplyInput.Blur()
			return nil
		}},
	}
}

func (m *Model) projectCommandItems(shortcuts commandShortcuts) []picker.CommandItem {
	return []picker.CommandItem{
		{Label: "Project: Open project picker", Desc: "Scan project roots and create session", Shortcut: shortcuts.openProject, Run: func() tea.Cmd {
			m.openProjectPicker()
			return nil
		}},
		{Label: "Project: Close project", Desc: "Hide project from tabs (sessions stay running)", Shortcut: shortcuts.closeProject, Run: func() tea.Cmd {
			m.openCloseProjectConfirm()
			return nil
		}},
		{Label: "Project: Close all projects", Desc: "Hide all projects from tabs (sessions stay running)", Run: func() tea.Cmd {
			m.openCloseAllProjectsConfirm()
			return nil
		}},
	}
}

func (m *Model) menuCommandItems() []picker.CommandItem {
	return []picker.CommandItem{
		{Label: "Settings", Desc: "Project roots and config", Run: func() tea.Cmd {
			return m.openSettingsMenu()
		}},
		{Label: "Debug", Desc: "Refresh and restart server", Run: func() tea.Cmd {
			return m.openDebugMenu()
		}},
	}
}

func (m *Model) otherCommandItems(shortcuts commandShortcuts) []picker.CommandItem {
	return []picker.CommandItem{
		{Label: "Other: Help", Desc: "Show help", Shortcut: shortcuts.help, Run: func() tea.Cmd {
			m.setState(StateHelp)
			return nil
		}},
		{Label: "Other: Quit", Desc: "Exit PeakyPanes", Shortcut: shortcuts.quit, Run: func() tea.Cmd {
			return tea.Quit
		}},
	}
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
