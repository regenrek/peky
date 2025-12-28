package app

import (
	"github.com/regenrek/peakypanes/internal/sessiond"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all incoming messages and returns the updated model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := m.handleUpdateMsg(msg); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handlePassiveUpdates(msg); handled {
		return model, cmd
	}
	return m, nil
}

func (m *Model) handleUpdateMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyWindowSize(msg)
		return m, nil, true
	case refreshTickMsg:
		return m, m.handleRefreshTick(msg), true
	case selectionRefreshMsg:
		return m, m.handleSelectionRefresh(msg), true
	case dashboardSnapshotMsg:
		return m, m.handleDashboardSnapshot(msg), true
	case daemonEventMsg:
		return m, m.handleDaemonEvent(msg), true
	case paneViewsMsg:
		m.handlePaneViews(msg)
		return m, nil, true
	case sessionStartedMsg:
		return m, m.handleSessionStarted(msg), true
	case SuccessMsg:
		m.setToast(msg.Message, toastSuccess)
		return m, nil, true
	case WarningMsg:
		m.setToast(msg.Message, toastWarning)
		return m, nil, true
	case InfoMsg:
		m.setToast(msg.Message, toastInfo)
		return m, nil, true
	case ErrorMsg:
		m.setToast(msg.Error(), toastError)
		return m, nil, true
	case tea.MouseMsg:
		return m.handleMouseMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	default:
		return nil, nil, false
	}
}

func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != StateDashboard {
		return nil, nil, false
	}
	model, cmd := m.updateDashboardMouse(msg)
	return model, cmd, true
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch m.state {
	case StateDashboard:
		model, cmd := m.updateDashboard(msg)
		return model, cmd, true
	case StateProjectPicker:
		model, cmd := m.updateProjectPicker(msg)
		return model, cmd, true
	case StateLayoutPicker:
		model, cmd := m.updateLayoutPicker(msg)
		return model, cmd, true
	case StatePaneSplitPicker:
		model, cmd := m.updatePaneSplitPicker(msg)
		return model, cmd, true
	case StatePaneSwapPicker:
		model, cmd := m.updatePaneSwapPicker(msg)
		return model, cmd, true
	case StateConfirmKill:
		model, cmd := m.updateConfirmKill(msg)
		return model, cmd, true
	case StateConfirmCloseProject:
		model, cmd := m.updateConfirmCloseProject(msg)
		return model, cmd, true
	case StateConfirmClosePane:
		model, cmd := m.updateConfirmClosePane(msg)
		return model, cmd, true
	case StateHelp:
		model, cmd := m.updateHelp(msg)
		return model, cmd, true
	case StateCommandPalette:
		model, cmd := m.updateCommandPalette(msg)
		return model, cmd, true
	case StateRenameSession, StateRenamePane:
		model, cmd := m.updateRename(msg)
		return model, cmd, true
	case StateProjectRootSetup:
		model, cmd := m.updateProjectRootSetup(msg)
		return model, cmd, true
	default:
		return nil, nil, false
	}
}

func (m *Model) handlePassiveUpdates(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if model, cmd, handled := m.handlePickerUpdate(msg); handled {
		return model, cmd, true
	}
	if m.filterActive {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd, true
	}
	return nil, nil, false
}

func (m *Model) handlePickerUpdate(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch m.state {
	case StateProjectPicker:
		var cmd tea.Cmd
		m.projectPicker, cmd = m.projectPicker.Update(msg)
		return m, cmd, true
	case StateLayoutPicker:
		var cmd tea.Cmd
		m.layoutPicker, cmd = m.layoutPicker.Update(msg)
		return m, cmd, true
	case StatePaneSwapPicker:
		var cmd tea.Cmd
		m.paneSwapPicker, cmd = m.paneSwapPicker.Update(msg)
		return m, cmd, true
	case StateCommandPalette:
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
		return m, cmd, true
	default:
		return nil, nil, false
	}
}

func (m *Model) applyWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.projectPicker.SetSize(msg.Width-4, msg.Height-4)
	m.setLayoutPickerSize()
	m.setPaneSwapPickerSize()
	m.setCommandPaletteSize()
	m.setQuickReplySize()
}

func (m *Model) handleRefreshTick(msg refreshTickMsg) tea.Cmd {
	if m.refreshInFlight == 0 {
		return tea.Batch(m.startRefreshCmd(), tickCmd(m.settings.RefreshInterval))
	}
	return tickCmd(m.settings.RefreshInterval)
}

func (m *Model) handleSelectionRefresh(msg selectionRefreshMsg) tea.Cmd {
	if msg.Version != m.selectionVersion {
		return nil
	}
	return m.startRefreshCmd()
}

func (m *Model) handleDashboardSnapshot(msg dashboardSnapshotMsg) tea.Cmd {
	m.endRefresh()
	if msg.Result.RefreshSeq > 0 && msg.Result.RefreshSeq < m.refreshSeq {
		return nil
	}
	if msg.Result.RefreshSeq < m.lastAppliedSeq {
		return nil
	}
	if msg.Result.Err != nil {
		m.setToast("Refresh failed: "+msg.Result.Err.Error(), toastError)
		return nil
	}
	m.lastAppliedSeq = msg.Result.RefreshSeq
	if msg.Result.Warning != "" {
		m.setToast("Dashboard config: "+msg.Result.Warning, toastWarning)
	}
	m.data = msg.Result.Data
	m.settings = msg.Result.Settings
	m.config = msg.Result.RawConfig
	if msg.Result.Keymap != nil {
		m.keys = msg.Result.Keymap
	}
	if msg.Result.Version == m.selectionVersion {
		m.applySelection(msg.Result.Resolved)
	} else {
		m.applySelection(resolveSelection(m.data.Projects, m.selection))
	}
	m.syncExpandedSessions()
	if m.refreshSelectionForProjectConfig() {
		m.setToast("Project config changed: selection refreshed", toastInfo)
	}
	return m.refreshPaneViewsCmd()
}

func (m *Model) handleDaemonEvent(msg daemonEventMsg) tea.Cmd {
	cmds := []tea.Cmd{waitDaemonEvent(m.client)}
	switch msg.Event.Type {
	case sessiond.EventPaneUpdated:
		if cmd := m.refreshPaneViewFor(msg.Event.PaneID); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case sessiond.EventSessionChanged:
		if m.refreshInFlight == 0 {
			cmds = append(cmds, m.startRefreshCmd())
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) handlePaneViews(msg paneViewsMsg) {
	if msg.Err != nil {
		m.setToast("Pane view failed: "+msg.Err.Error(), toastWarning)
	}
	for _, view := range msg.Views {
		key := paneViewKeyFrom(view)
		m.paneViews[key] = view.View
		if view.PaneID != "" {
			m.paneMouseMotion[view.PaneID] = view.AllowMotion
		}
	}
}

func (m *Model) handleSessionStarted(msg sessionStartedMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Start failed: "+msg.Err.Error(), toastError)
		return nil
	}
	if msg.Name != "" {
		m.setToast("Session started: "+msg.Name, toastSuccess)
		projectName := m.projectNameForPath(msg.Path)
		if projectName != "" {
			m.selection.Project = projectName
		}
		m.selection.Session = msg.Name
		m.selection.Pane = ""
		m.selectionVersion++
		m.rememberSelection(m.selection)
	} else {
		m.setToast("Session started", toastSuccess)
	}
	m.setTerminalFocus(msg.Focus)
	return m.startRefreshCmd()
}
