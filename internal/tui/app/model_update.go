package app

import (
	"context"
	"errors"
	"time"

	"github.com/regenrek/peakypanes/internal/diag"
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
		return m, m.handlePaneViews(msg), true
	case daemonRestartMsg:
		return m, m.handleDaemonRestart(msg), true
	case PaneClosedMsg:
		return m, m.handlePaneClosed(msg), true
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
	case StateConfirmCloseAllProjects:
		model, cmd := m.updateConfirmCloseAllProjects(msg)
		return model, cmd, true
	case StateConfirmClosePane:
		model, cmd := m.updateConfirmClosePane(msg)
		return model, cmd, true
	case StateConfirmRestart:
		model, cmd := m.updateConfirmRestart(msg)
		return model, cmd, true
	case StateHelp:
		model, cmd := m.updateHelp(msg)
		return model, cmd, true
	case StateCommandPalette:
		model, cmd := m.updateCommandPalette(msg)
		return model, cmd, true
	case StateSettingsMenu:
		model, cmd := m.updateSettingsMenu(msg)
		return model, cmd, true
	case StateDebugMenu:
		model, cmd := m.updateDebugMenu(msg)
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
	case StateSettingsMenu:
		var cmd tea.Cmd
		m.settingsMenu, cmd = m.settingsMenu.Update(msg)
		return m, cmd, true
	case StateDebugMenu:
		var cmd tea.Cmd
		m.debugMenu, cmd = m.debugMenu.Update(msg)
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
	m.setSettingsMenuSize()
	m.setDebugMenuSize()
	m.setQuickReplySize()
}

func (m *Model) handleRefreshTick(msg refreshTickMsg) tea.Cmd {
	if m.refreshInFlight == 0 {
		diag.LogEvery("tui.refresh.start", 2*time.Second, "tui: refresh tick start next_seq=%d", m.refreshSeq+1)
		return tea.Batch(m.startRefreshCmd(), tickCmd(m.settings.RefreshInterval))
	}
	diag.LogEvery("tui.refresh.skip", 2*time.Second, "tui: refresh tick skipped in_flight=%d seq=%d", m.refreshInFlight, m.refreshSeq)
	return tickCmd(m.settings.RefreshInterval)
}

func (m *Model) handleSelectionRefresh(msg selectionRefreshMsg) tea.Cmd {
	if msg.Version != m.selectionVersion {
		return nil
	}
	return m.requestRefreshCmd()
}

func (m *Model) handleDashboardSnapshot(msg dashboardSnapshotMsg) tea.Cmd {
	m.endRefresh()
	diag.LogEvery("tui.snapshot.recv", 2*time.Second, "tui: snapshot recv seq=%d err=%v in_flight=%d", msg.Result.RefreshSeq, msg.Result.Err, m.refreshInFlight)
	if msg.Result.RefreshSeq > 0 && msg.Result.RefreshSeq < m.refreshSeq {
		if m.refreshQueued {
			m.refreshQueued = false
			return m.startRefreshCmd()
		}
		return nil
	}
	if msg.Result.RefreshSeq < m.lastAppliedSeq {
		if m.refreshQueued {
			m.refreshQueued = false
			return m.startRefreshCmd()
		}
		return nil
	}
	if msg.Result.Err != nil {
		m.setToast("Refresh failed: "+msg.Result.Err.Error(), toastError)
		cmd := m.refreshPaneViewsCmd()
		if m.refreshQueued {
			m.refreshQueued = false
			return tea.Batch(cmd, m.startRefreshCmd())
		}
		return cmd
	}
	m.lastAppliedSeq = msg.Result.RefreshSeq
	if msg.Result.Warning != "" {
		m.setToast("Dashboard config: "+msg.Result.Warning, toastWarning)
	}
	m.data = msg.Result.Data
	m.reconcilePaneInputDisabled()
	m.settings = msg.Result.Settings
	m.config = msg.Result.RawConfig
	if msg.Result.Keymap != nil {
		m.keys = msg.Result.Keymap
	}
	if msg.Result.Version == m.selectionVersion {
		m.applySelection(msg.Result.Resolved)
	} else {
		m.applySelection(resolveSelectionForTab(m.tab, m.data.Projects, m.selection))
	}
	m.syncExpandedSessions()
	if m.refreshSelectionForProjectConfig() {
		m.setToast("Project config changed: selection refreshed", toastInfo)
	}
	cmd := m.refreshPaneViewsCmd()
	if m.refreshQueued {
		m.refreshQueued = false
		cmd = tea.Batch(cmd, m.startRefreshCmd())
	}
	return cmd
}

func (m *Model) handlePaneClosed(msg PaneClosedMsg) tea.Cmd {
	if msg.PaneID != "" {
		if m.isPaneInputDisabled(msg.PaneID) {
			return nil
		}
		m.markPaneInputDisabled(msg.PaneID)
	}
	if msg.Message != "" {
		m.setToast(msg.Message, toastWarning)
	} else {
		m.setToast("Pane closed", toastWarning)
	}
	return m.requestRefreshCmd()
}

const daemonEventBatchMax = 64

func (m *Model) handleDaemonEvent(msg daemonEventMsg) tea.Cmd {
	events := []sessiond.Event{msg.Event}
	if m != nil && m.client != nil {
		ch := m.client.Events()
		drain := true
		for i := 0; i < daemonEventBatchMax && drain; i++ {
			select {
			case evt, ok := <-ch:
				if !ok {
					drain = false
					break
				}
				events = append(events, evt)
			default:
				drain = false
			}
		}
	}
	paneIDs := make(map[string]struct{})
	refresh := false
	for _, event := range events {
		switch event.Type {
		case sessiond.EventPaneUpdated:
			if event.PaneID != "" {
				paneIDs[event.PaneID] = struct{}{}
			}
		case sessiond.EventSessionChanged:
			refresh = true
		}
	}
	diag.LogEvery("tui.event", 2*time.Second, "tui: events batch=%d panes=%d refresh=%v", len(events), len(paneIDs), refresh)
	cmds := []tea.Cmd{waitDaemonEvent(m.client)}
	if refresh {
		if cmd := m.requestRefreshCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(paneIDs) > 0 {
		if cmd := m.refreshPaneViewsForIDs(paneIDs); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) handlePaneViews(msg paneViewsMsg) tea.Cmd {
	var cmd tea.Cmd
	diag.LogEvery("tui.paneviews.recv", 2*time.Second, "tui: paneViews recv count=%d err=%v in_flight=%d queued=%v queued_ids=%d", len(msg.Views), msg.Err, m.paneViewInFlight, m.paneViewQueued, len(m.paneViewQueuedIDs))
	if msg.Err != nil && len(msg.Views) == 0 && !errors.Is(msg.Err, context.DeadlineExceeded) && !errors.Is(msg.Err, context.Canceled) {
		m.setToast("Pane view failed: "+msg.Err.Error(), toastWarning)
	}
	if m.paneViews == nil {
		m.paneViews = make(map[paneViewKey]string)
	}
	if m.paneMouseMotion == nil {
		m.paneMouseMotion = make(map[string]bool)
	}
	if m.paneViewSeq == nil {
		m.paneViewSeq = make(map[paneViewKey]uint64)
	}
	for _, view := range msg.Views {
		key := paneViewKeyFrom(view)

		if view.UpdateSeq > 0 {
			m.paneViewSeq[key] = view.UpdateSeq
		}
		if !view.NotModified {
			if view.View != "" {
				m.paneViews[key] = view.View
			}
		}

		if view.PaneID != "" {
			m.paneMouseMotion[view.PaneID] = view.AllowMotion
		}
	}
	if m.paneViewInFlight > 0 {
		m.paneViewInFlight--
	}
	if m.paneViewInFlight == 0 {
		if m.paneViewQueued {
			m.paneViewQueued = false
			m.paneViewQueuedIDs = nil
			cmd = m.refreshPaneViewsCmd()
		} else if len(m.paneViewQueuedIDs) > 0 {
			queued := m.paneViewQueuedIDs
			m.paneViewQueuedIDs = nil
			cmd = m.startPaneViewFetch(m.paneViewRequestsForIDs(queued))
		}
	}
	return cmd
}

func (m *Model) handleSessionStarted(msg sessionStartedMsg) tea.Cmd {
	if msg.Err != nil {
		m.setToast("Start failed: "+msg.Err.Error(), toastError)
		return nil
	}
	if msg.Name != "" {
		m.setToast("Session started: "+msg.Name, toastSuccess)
		projectName := m.projectNameForPath(msg.Path)
		projectID := projectKey(msg.Path, projectName)
		if projectID != "" {
			m.selection.ProjectID = projectID
		}
		m.selection.Session = msg.Name
		m.selection.Pane = ""
		m.selectionVersion++
		m.rememberSelection(m.selection)
	} else {
		m.setToast("Session started", toastSuccess)
	}
	m.setTerminalFocus(msg.Focus)
	return m.requestRefreshCmd()
}
