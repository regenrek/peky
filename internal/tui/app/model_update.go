package app

import (
	"context"
	"errors"
	"reflect"
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
	if handler, ok := updateHandlers[reflect.TypeOf(msg)]; ok {
		model, cmd := handler(m, msg)
		return model, cmd, true
	}
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouseMsg(mouseMsg)
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyMsg(keyMsg)
	}
	return nil, nil, false
}

func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != StateDashboard {
		return nil, nil, false
	}
	model, cmd := m.updateDashboardMouse(msg)
	return model, cmd, true
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if handler := keyHandlers[m.state]; handler != nil {
		model, cmd := handler(m, msg)
		return model, cmd, true
	}
	return nil, nil, false
}

type keyHandler func(*Model, tea.KeyMsg) (tea.Model, tea.Cmd)

var keyHandlers = map[ViewState]keyHandler{
	StateDashboard:       func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateDashboard(msg) },
	StateProjectPicker:   func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateProjectPicker(msg) },
	StateLayoutPicker:    func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateLayoutPicker(msg) },
	StatePaneSplitPicker: func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updatePaneSplitPicker(msg) },
	StatePaneSwapPicker:  func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updatePaneSwapPicker(msg) },
	StateConfirmKill:     func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateConfirmKill(msg) },
	StateConfirmCloseProject: func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return m.updateConfirmCloseProject(msg)
	},
	StateConfirmCloseAllProjects: func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return m.updateConfirmCloseAllProjects(msg)
	},
	StateConfirmClosePane: func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateConfirmClosePane(msg) },
	StateConfirmRestart:   func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateConfirmRestart(msg) },
	StateConfirmQuit:      func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateConfirmQuit(msg) },
	StateHelp:             func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateHelp(msg) },
	StateCommandPalette:   func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateCommandPalette(msg) },
	StateSettingsMenu:     func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateSettingsMenu(msg) },
	StateDebugMenu:        func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateDebugMenu(msg) },
	StateRenameSession:    func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateRename(msg) },
	StateRenamePane:       func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateRename(msg) },
	StateProjectRootSetup: func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateProjectRootSetup(msg) },
}

type updateHandler func(*Model, tea.Msg) (tea.Model, tea.Cmd)

var updateHandlers = map[reflect.Type]updateHandler{
	reflect.TypeOf(tea.WindowSizeMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		m.applyWindowSize(msg.(tea.WindowSizeMsg))
		return m, nil
	},
	reflect.TypeOf(refreshTickMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleRefreshTick(msg.(refreshTickMsg))
	},
	reflect.TypeOf(selectionRefreshMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleSelectionRefresh(msg.(selectionRefreshMsg))
	},
	reflect.TypeOf(dashboardSnapshotMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleDashboardSnapshot(msg.(dashboardSnapshotMsg))
	},
	reflect.TypeOf(daemonEventMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleDaemonEvent(msg.(daemonEventMsg))
	},
	reflect.TypeOf(paneViewsMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePaneViews(msg.(paneViewsMsg))
	},
	reflect.TypeOf(paneViewPumpMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePaneViewPump(msg.(paneViewPumpMsg))
	},
	reflect.TypeOf(daemonRestartMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleDaemonRestart(msg.(daemonRestartMsg))
	},
	reflect.TypeOf(daemonStopMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleDaemonStop(msg.(daemonStopMsg))
	},
	reflect.TypeOf(PaneClosedMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePaneClosed(msg.(PaneClosedMsg))
	},
	reflect.TypeOf(quickReplySendMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleQuickReplySend(msg.(quickReplySendMsg))
	},
	reflect.TypeOf(sessionStartedMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleSessionStarted(msg.(sessionStartedMsg))
	},
	reflect.TypeOf(SuccessMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		m.setToast(msg.(SuccessMsg).Message, toastSuccess)
		return m, nil
	},
	reflect.TypeOf(WarningMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		m.setToast(msg.(WarningMsg).Message, toastWarning)
		return m, nil
	},
	reflect.TypeOf(InfoMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		m.setToast(msg.(InfoMsg).Message, toastInfo)
		return m, nil
	},
	reflect.TypeOf(ErrorMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		m.setToast(msg.(ErrorMsg).Error(), toastError)
		return m, nil
	},
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
	m.noteRefreshTiming(msg)
	if cmd, handled := m.handleSnapshotStale(msg); handled {
		return cmd
	}
	if msg.Result.Err != nil {
		return m.handleSnapshotError(msg)
	}
	m.applySnapshotState(msg)
	return m.handleSnapshotPostApply()
}

func (m *Model) noteRefreshTiming(msg dashboardSnapshotMsg) {
	if !perfDebugEnabled() || msg.Result.RefreshSeq == 0 {
		return
	}
	if m.refreshStarted == nil {
		return
	}
	started, ok := m.refreshStarted[msg.Result.RefreshSeq]
	if !ok {
		return
	}
	delete(m.refreshStarted, msg.Result.RefreshSeq)
	dur := time.Since(started)
	if dur > perfSlowRefreshTotal {
		logPerfEvery("tui.refresh.total", perfLogInterval, "tui: refresh slow seq=%d dur=%s err=%v", msg.Result.RefreshSeq, dur, msg.Result.Err)
	}
}

func (m *Model) handleSnapshotStale(msg dashboardSnapshotMsg) (tea.Cmd, bool) {
	if msg.Result.RefreshSeq > 0 && msg.Result.RefreshSeq < m.refreshSeq {
		return m.finishQueuedRefresh(), true
	}
	if msg.Result.RefreshSeq < m.lastAppliedSeq {
		return m.finishQueuedRefresh(), true
	}
	return nil, false
}

func (m *Model) finishQueuedRefresh() tea.Cmd {
	if !m.refreshQueued {
		return nil
	}
	m.refreshQueued = false
	return m.startRefreshCmd()
}

func (m *Model) handleSnapshotError(msg dashboardSnapshotMsg) tea.Cmd {
	m.setToast("Refresh failed: "+msg.Result.Err.Error(), toastError)
	return m.appendQueuedRefresh(m.refreshPaneViewsCmd())
}

func (m *Model) applySnapshotState(msg dashboardSnapshotMsg) {
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
}

func (m *Model) handleSnapshotPostApply() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 2)
	if pumpCmd := m.snapshotPumpCmd(); pumpCmd != nil {
		cmds = append(cmds, pumpCmd)
	}
	if refreshCmd := m.refreshPaneViewsCmd(); refreshCmd != nil {
		cmds = append(cmds, refreshCmd)
	}
	if cmd := m.appendQueuedRefresh(nil); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if len(cmds) == 0 {
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

func (m *Model) snapshotPumpCmd() tea.Cmd {
	if len(m.paneViewQueuedIDs) == 0 {
		return nil
	}
	return m.schedulePaneViewPump("snapshot_pending", 0)
}

func (m *Model) appendQueuedRefresh(cmd tea.Cmd) tea.Cmd {
	if !m.refreshQueued {
		return cmd
	}
	m.refreshQueued = false
	if cmd == nil {
		return m.startRefreshCmd()
	}
	return tea.Batch(cmd, m.startRefreshCmd())
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
	events := collectDaemonEvents(m, msg.Event)
	if perfDebugEnabled() {
		now := time.Now()
		for _, event := range events {
			if event.Type == sessiond.EventPaneUpdated && event.PaneID != "" {
				m.perfNotePaneUpdated(event.PaneID, event.PaneUpdateSeq, now)
			}
		}
	}
	paneIDs, refresh, toastMsg, toastLevel := summarizeDaemonEvents(events)
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
	if toastMsg != "" {
		m.setToast(toastMsg, toastLevel)
	}
	return tea.Batch(cmds...)
}

func collectDaemonEvents(m *Model, first sessiond.Event) []sessiond.Event {
	events := []sessiond.Event{first}
	if m == nil || m.client == nil {
		return events
	}
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
	return events
}

func summarizeDaemonEvents(events []sessiond.Event) (map[string]struct{}, bool, string, toastLevel) {
	paneIDs := make(map[string]struct{})
	refresh := false
	toastMsg := ""
	toastLevel := toastInfo
	for _, event := range events {
		switch event.Type {
		case sessiond.EventPaneUpdated:
			if event.PaneID != "" {
				paneIDs[event.PaneID] = struct{}{}
			}
		case sessiond.EventSessionChanged:
			refresh = true
		case sessiond.EventToast:
			if event.Toast != "" {
				toastMsg = event.Toast
				toastLevel = toastLevelFromSessiond(event.ToastKind)
			}
		}
	}
	return paneIDs, refresh, toastMsg, toastLevel
}

func toastLevelFromSessiond(level sessiond.ToastLevel) toastLevel {
	switch level {
	case sessiond.ToastSuccess:
		return toastSuccess
	case sessiond.ToastWarning:
		return toastWarning
	case sessiond.ToastInfo:
		return toastInfo
	default:
		return toastInfo
	}
}

func (m *Model) handlePaneViews(msg paneViewsMsg) tea.Cmd {
	var cmd tea.Cmd
	diag.LogEvery("tui.paneviews.recv", 2*time.Second, "tui: paneViews recv count=%d err=%v in_flight=%d pending=%d", len(msg.Views), msg.Err, m.paneViewInFlight, len(m.paneViewQueuedIDs))
	if msg.Err != nil && len(msg.Views) == 0 && !errors.Is(msg.Err, context.DeadlineExceeded) && !errors.Is(msg.Err, context.Canceled) {
		m.setToast("Pane view failed: "+msg.Err.Error(), toastWarning)
	}
	m.ensurePaneViewMaps()
	for _, view := range msg.Views {
		m.applyPaneView(view)
	}
	m.clearPaneViewInFlight(msg.PaneIDs)
	if m.paneViewInFlight > 0 {
		m.paneViewInFlight--
	}
	if len(m.paneViewQueuedIDs) > 0 {
		cmd = m.schedulePaneViewPump("pane_view_done", 0)
	}
	return cmd
}

func (m *Model) ensurePaneViewMaps() {
	if m.paneViews == nil {
		m.paneViews = make(map[paneViewKey]string)
	}
	if m.paneMouseMotion == nil {
		m.paneMouseMotion = make(map[string]bool)
	}
	if m.paneViewSeq == nil {
		m.paneViewSeq = make(map[paneViewKey]uint64)
	}
}

func (m *Model) applyPaneView(view sessiond.PaneViewResponse) {
	key := paneViewKeyFrom(view)
	if view.UpdateSeq > 0 {
		m.paneViewSeq[key] = view.UpdateSeq
	}
	if view.PaneID != "" && view.Cols > 0 && view.Rows > 0 {
		m.recordPaneSize(view.PaneID, view.Cols, view.Rows)
	}
	if !view.NotModified && view.View != "" {
		m.paneViews[key] = view.View
	}
	if view.PaneID != "" {
		m.paneMouseMotion[view.PaneID] = view.AllowMotion
	}
	if perfDebugEnabled() && view.PaneID != "" && view.View != "" {
		if m.paneViewFirst == nil {
			m.paneViewFirst = make(map[string]struct{})
		}
		if _, ok := m.paneViewFirst[view.PaneID]; !ok {
			m.paneViewFirst[view.PaneID] = struct{}{}
			logPerfEvery("tui.paneview.first."+view.PaneID, 0, "tui: pane view first pane=%s mode=%v cols=%d rows=%d", view.PaneID, view.Mode, view.Cols, view.Rows)
		}
	}
	m.perfNotePaneViewResponse(view, time.Now())
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
