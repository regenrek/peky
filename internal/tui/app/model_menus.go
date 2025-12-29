package app

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

const settingsMenuHeading = "Settings"
const debugMenuHeading = "Debug"

func (m *Model) setupSettingsMenu() {
	m.settingsMenu = picker.NewDialogMenu()
}

func (m *Model) setupDebugMenu() {
	m.debugMenu = picker.NewDialogMenu()
}

func (m *Model) openSettingsMenu() tea.Cmd {
	cmd := m.settingsMenu.SetItems(m.settingsMenuItems())
	m.setSettingsMenuSize()
	m.setState(StateSettingsMenu)
	return cmd
}

func (m *Model) openDebugMenu() tea.Cmd {
	cmd := m.debugMenu.SetItems(m.debugMenuItems())
	m.setDebugMenuSize()
	m.setState(StateDebugMenu)
	return cmd
}

func (m *Model) updateSettingsMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		return m, m.runSettingsMenuSelection()
	}

	var cmd tea.Cmd
	m.settingsMenu, cmd = m.settingsMenu.Update(msg)
	return m, cmd
}

func (m *Model) updateDebugMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "enter":
		return m, m.runDebugMenuSelection()
	}

	var cmd tea.Cmd
	m.debugMenu, cmd = m.debugMenu.Update(msg)
	return m, cmd
}

func (m *Model) runSettingsMenuSelection() tea.Cmd {
	item, ok := m.settingsMenu.SelectedItem().(picker.CommandItem)
	m.setState(StateDashboard)
	if !ok || item.Run == nil {
		return nil
	}
	return item.Run()
}

func (m *Model) runDebugMenuSelection() tea.Cmd {
	item, ok := m.debugMenu.SelectedItem().(picker.CommandItem)
	m.setState(StateDashboard)
	if !ok || item.Run == nil {
		return nil
	}
	return item.Run()
}

func (m *Model) settingsMenuItems() []list.Item {
	shortcut := ""
	if m.keys != nil {
		shortcut = keyLabel(m.keys.editConfig)
	}
	items := []picker.CommandItem{
		{Label: "Set project directory", Desc: "Choose folders to scan for projects", Run: func() tea.Cmd {
			m.openProjectRootSetup()
			return nil
		}},
		{Label: "Edit global config", Desc: "Open config in $EDITOR", Shortcut: shortcut, Run: func() tea.Cmd {
			return m.editConfig()
		}},
	}
	return commandItemsToList(items)
}

func (m *Model) debugMenuItems() []list.Item {
	shortcut := ""
	if m.keys != nil {
		shortcut = keyLabel(m.keys.refresh)
	}
	items := []picker.CommandItem{
		{Label: "Refresh", Desc: "Refresh dashboard data", Shortcut: shortcut, Run: func() tea.Cmd {
			m.setToast("Refreshing...", toastInfo)
			return m.requestRefreshCmd()
		}},
		{Label: "Restart Server", Desc: "Restart server and restore sessions", Run: func() tea.Cmd {
			m.openRestartConfirm()
			return nil
		}},
	}
	return commandItemsToList(items)
}

func (m *Model) setSettingsMenuSize() {
	m.setDialogMenuSize(&m.settingsMenu, settingsMenuHeading, 34, 64, 8, 16)
}

func (m *Model) setDebugMenuSize() {
	m.setDialogMenuSize(&m.debugMenu, debugMenuHeading, 34, 64, 8, 16)
}

func (m *Model) setDialogMenuSize(menu *list.Model, heading string, minW, maxW, minH, maxH int) {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	headerHeight := lipgloss.Height(theme.HelpTitle.Render(heading))
	hFrame, vFrame := dialogStyleCompact.GetFrameSize()
	availableW := m.width - 6
	availableH := m.height - 4
	if availableW < 30 {
		availableW = m.width
	}
	if availableH < 10 {
		availableH = m.height
	}
	desiredW := clamp(availableW, minW, maxW)
	desiredH := clamp(availableH, minH, maxH)
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
	if listH < 4 {
		if maxListH < 4 {
			listH = maxListH
		} else {
			listH = 4
		}
	}
	menu.SetSize(listW, listH)
}
