package app

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/picker"
)

// ===== Command palette =====

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
	hFrame, vFrame := dialogStyle.GetFrameSize()
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
	listH := desiredH - vFrame
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	if listH < 6 {
		listH = clamp(m.height-vFrame, 6, m.height)
	}
	m.commandPalette.SetSize(listW, listH)
}

func (m *Model) commandPaletteItems() []list.Item {
	items := []picker.CommandItem{
		{Label: "Project: Open project picker", Desc: "Scan project roots and create session", Run: func() tea.Cmd {
			m.openProjectPicker()
			return nil
		}},
		{Label: "Project: Set project roots", Desc: "Choose folders to scan for projects", Run: func() tea.Cmd {
			m.openProjectRootSetup()
			return nil
		}},
		{Label: "Project: Close project", Desc: "Hide project from tabs (sessions stay running)", Run: func() tea.Cmd {
			m.openCloseProjectConfirm()
			return nil
		}},
	}
	for _, entry := range m.hiddenProjectEntries() {
		entry := entry
		label := hiddenProjectLabel(entry)
		items = append(items, picker.CommandItem{
			Label: "Project: Reopen " + label,
			Desc:  "Show hidden project",
			Run: func() tea.Cmd {
				return m.reopenHiddenProject(entry)
			},
		})
	}
	items = append(items, []picker.CommandItem{
		{Label: "Session: Attach / start", Desc: "Attach to running session or start if stopped", Run: func() tea.Cmd {
			return m.attachOrStart()
		}},
		{Label: "Session: New session", Desc: "Pick a layout and create a new session", Run: func() tea.Cmd {
			m.openLayoutPicker()
			return nil
		}},
		{Label: "Session: Kill session", Desc: "Kill the selected session", Run: func() tea.Cmd {
			m.openKillConfirm()
			return nil
		}},
		{Label: "Pane: Add pane", Desc: "Split the selected pane", Run: func() tea.Cmd {
			m.openPaneSplitPicker()
			return nil
		}},
		{Label: "Pane: Swap pane", Desc: "Swap the selected pane with another", Run: func() tea.Cmd {
			m.openPaneSwapPicker()
			return nil
		}},
		{Label: "Pane: Close pane", Desc: "Close the selected pane", Run: func() tea.Cmd {
			return m.openClosePaneConfirm()
		}},
		{Label: "Pane: Quick reply", Desc: "Send a short follow-up to the selected pane", Run: func() tea.Cmd {
			return m.openQuickReply()
		}},
		{Label: "Pane: Rename pane", Desc: "Rename the selected pane title", Run: func() tea.Cmd {
			m.openRenamePane()
			return nil
		}},
		{Label: "Pane: Toggle pane list", Desc: "Expand/collapse pane list", Run: func() tea.Cmd {
			m.togglePanes()
			return nil
		}},
		{Label: "Session: Rename session", Desc: "Rename the selected session", Run: func() tea.Cmd {
			m.openRenameSession()
			return nil
		}},
		{Label: "Other: Refresh", Desc: "Refresh dashboard data", Run: func() tea.Cmd {
			m.setToast("Refreshing...", toastInfo)
			m.refreshing = true
			return m.refreshCmd()
		}},
		{Label: "Other: Edit config", Desc: "Open config in $EDITOR", Run: func() tea.Cmd {
			return m.editConfig()
		}},
		{Label: "Other: Filter sessions", Desc: "Filter session list", Run: func() tea.Cmd {
			m.filterActive = true
			m.filterInput.Focus()
			m.quickReplyInput.Blur()
			return nil
		}},
		{Label: "Other: Help", Desc: "Show help", Run: func() tea.Cmd {
			m.setState(StateHelp)
			return nil
		}},
		{Label: "Other: Quit", Desc: "Exit PeakyPanes", Run: func() tea.Cmd {
			return tea.Quit
		}},
	}...)
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func (m *Model) updateCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commandPalette.FilterState() == list.Filtering {
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
		item, ok := m.commandPalette.SelectedItem().(picker.CommandItem)
		m.commandPalette.ResetFilter()
		m.setState(StateDashboard)
		if !ok || item.Run == nil {
			return m, nil
		}
		return m, item.Run()
	}

	var cmd tea.Cmd
	m.commandPalette, cmd = m.commandPalette.Update(msg)
	return m, cmd
}
