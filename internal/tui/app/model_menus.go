package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

const settingsMenuHeading = "Settings"
const debugMenuHeading = "Debug"
const performanceMenuHelpFooterLines = 5

func (m *Model) setupSettingsMenu() {
	m.settingsMenu = picker.NewDialogMenu()
}

func (m *Model) setupPerformanceMenu() {
	m.perfMenu = picker.NewDialogMenu()
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

func (m *Model) openPerformanceMenu() tea.Cmd {
	cmd := m.perfMenu.SetItems(m.performanceMenuItems())
	m.setPerformanceMenuSize()
	m.dialogHelpOpen = false
	m.perfMenu.SetShowHelp(true)
	m.perfMenu.KeyMap.ShowFullHelp = key.NewBinding(key.WithKeys(""), key.WithHelp("", ""))
	m.setState(StatePerformanceMenu)
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

func (m *Model) updatePerformanceMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.dialogHelpOpen = false
		return m, m.openSettingsMenu()
	case "enter":
		return m, m.runPerformanceMenuSelection()
	case "?":
		m.dialogHelpOpen = !m.dialogHelpOpen
		return m, nil
	default:
	}

	var cmd tea.Cmd
	m.perfMenu, cmd = m.perfMenu.Update(msg)
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

func (m *Model) runPerformanceMenuSelection() tea.Cmd {
	item, ok := m.perfMenu.SelectedItem().(picker.CommandItem)
	if !ok || item.Run == nil {
		return nil
	}
	prevState := m.state
	cmd := item.Run()
	if m.state != prevState || m.state != StatePerformanceMenu {
		return cmd
	}
	refresh := m.perfMenu.SetItems(m.performanceMenuItems())
	if cmd == nil {
		return refresh
	}
	return tea.Batch(cmd, refresh)
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
		{Label: "Performance", Desc: "Tune live rendering and throttling", Run: func() tea.Cmd {
			return m.openPerformanceMenu()
		}},
		{Label: "Edit global config", Desc: "Open config in $EDITOR", Shortcut: shortcut, Run: func() tea.Cmd {
			return m.editConfig()
		}},
	}
	return commandItemsToList(items)
}

func (m *Model) performanceMenuItems() []list.Item {
	presetValue := titleCase(m.settings.Performance.Preset)
	presetHelpValue := strings.ToLower(m.settings.Performance.Preset)
	presetSeverity := dialogHelpSeverityFor(helpKeyPerfPreset, presetHelpValue)
	presetLabel := fmt.Sprintf("Preset: %s", dialogHelpValueStyle(presetSeverity, presetValue))

	renderValue := strings.ToLower(m.settings.Performance.RenderPolicy)
	renderSeverity := dialogHelpSeverityFor(helpKeyPerfRenderPolicy, renderValue)
	renderLabel := "Render policy: " + dialogHelpValueStyle(renderSeverity, renderValue)

	previewValue := strings.ToLower(m.settings.Performance.PreviewRender.Mode)
	previewSeverity := dialogHelpSeverityFor(helpKeyPerfPreviewRender, previewValue)
	previewLabel := "Preview render: " + dialogHelpValueStyle(previewSeverity, previewValue)
	presetDesc := "Low: battery saver • Medium: default • High: smoother • " + theme.FormatWarning("Max: no throttle")
	renderDesc := "Visible panes only (default) • " + theme.FormatWarning("All panes live")
	previewDesc := "Cached (default) • " + theme.FormatWarning("Direct: heavy CPU") + " • Off: disable previews"

	items := []picker.CommandItem{
		{Label: presetLabel, Desc: presetDesc, HelpKey: helpKeyPerfPreset, HelpValue: presetHelpValue, Run: func() tea.Cmd {
			return m.cyclePerformancePreset()
		}},
		{Label: renderLabel, Desc: renderDesc, HelpKey: helpKeyPerfRenderPolicy, HelpValue: renderValue, Run: func() tea.Cmd {
			return m.toggleRenderPolicy()
		}},
		{Label: previewLabel, Desc: previewDesc, HelpKey: helpKeyPerfPreviewRender, HelpValue: previewValue, Run: func() tea.Cmd {
			return m.cyclePreviewRenderMode()
		}},
		{Label: "Edit config (custom overrides)", Desc: "Open config in $EDITOR for fine tuning", HelpKey: helpKeyPerfEditConfig, Run: func() tea.Cmd {
			return m.editConfig()
		}},
		{Label: "Back", Desc: "Return to settings", Run: func() tea.Cmd {
			return m.openSettingsMenu()
		}},
	}
	return commandItemsToList(items)
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if len(value) == 1 {
		return strings.ToUpper(value)
	}
	return strings.ToUpper(value[:1]) + value[1:]
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
	m.setDialogMenuSize(&m.settingsMenu, settingsMenuHeading, 34, 64, 8, 16, 0)
}

func (m *Model) setPerformanceMenuSize() {
	m.setDialogMenuSize(&m.perfMenu, "Performance", 34, 64, 8, 16, performanceMenuHelpFooterLines)
}

func (m *Model) setDebugMenuSize() {
	m.setDialogMenuSize(&m.debugMenu, debugMenuHeading, 34, 64, 8, 16, 0)
}

func (m *Model) setDialogMenuSize(menu *list.Model, heading string, minW, maxW, minH, maxH, footerLines int) {
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
	listH := desiredH - vFrame - headerHeight - footerLines
	if listW < 20 {
		listW = clamp(m.width-hFrame, 20, m.width)
	}
	maxListH := m.height - vFrame - headerHeight - footerLines
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
