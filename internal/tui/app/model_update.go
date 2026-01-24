package app

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"github.com/regenrek/peakypanes/internal/logging"
	"github.com/regenrek/peakypanes/internal/sessiond"
	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all incoming messages and returns the updated model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := m.handleUpdateMsg(msg); handled {
		return m.appendFocusResult(model, cmd)
	}
	if model, cmd, handled := m.handlePassiveUpdates(msg); handled {
		return m.appendFocusResult(model, cmd)
	}
	return m, nil
}

func (m *Model) appendFocusResult(model tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if updated, ok := model.(*Model); ok && updated != nil {
		return updated, updated.appendFocusCmd(cmd)
	}
	if m != nil {
		return model, m.appendFocusCmd(cmd)
	}
	return model, cmd
}

func (m *Model) handleUpdateMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	m.maybeCancelPeky(msg)

	switch typed := msg.(type) {
	case tuiinput.KeyMsg:
		return m.handleInputKeyMsg(typed)
	case tea.MouseMsg:
		return m.handleMouseMsg(typed)
	case tea.KeyMsg:
		return m.handleKeyMsg(typed)
	default:
		if handler, ok := updateHandlers[reflect.TypeOf(msg)]; ok {
			model, cmd := handler(m, msg)
			return model, cmd, true
		}
		return nil, nil, false
	}
}

func (m *Model) maybeCancelPeky(msg tea.Msg) {
	if m == nil || !m.pekyBusy {
		return
	}
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "esc" {
			m.cancelPekyRun()
		}
	case tuiinput.KeyMsg:
		if typed.Tea().String() == "esc" {
			m.cancelPekyRun()
		}
	}
}

func (m *Model) handleInputKeyMsg(keyMsg tuiinput.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.keys != nil {
		if cmd, handled := m.handleGlobalInputKeyBindings(keyMsg); handled {
			return m, cmd, true
		}
	}
	if m.state == StateDashboard {
		model, cmd := m.updateDashboardInput(keyMsg)
		return model, cmd, true
	}
	return m.handleKeyMsg(keyMsg.Tea())
}

func (m *Model) handleGlobalInputKeyBindings(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	switch {
	case matchesBinding(msg, m.keys.help):
		m.setState(StateHelp)
		return nil, true
	case matchesBinding(msg, m.keys.quit):
		return m.requestQuit(), true
	case matchesBinding(msg, m.keys.commandPalette):
		return m.openCommandPalette(), true
	case matchesBinding(msg, m.keys.refresh):
		m.setToast("Refreshing...", toastInfo)
		return m.requestRefreshCmd(), true
	case matchesBinding(msg, m.keys.editConfig):
		return m.editConfig(), true
	default:
		return nil, false
	}
}

func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	switch m.state {
	case StateDashboard:
		model, cmd := m.updateDashboardMouse(msg)
		return model, cmd, true
	case StateProjectPicker:
		model, cmd := m.updateProjectPickerMouse(msg)
		return model, cmd, true
	default:
		return nil, nil, false
	}
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
	StatePerformanceMenu:  func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updatePerformanceMenu(msg) },
	StateDebugMenu:        func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateDebugMenu(msg) },
	StateRenameSession:    func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateRename(msg) },
	StateRenamePane:       func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateRename(msg) },
	StatePaneColor:        func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updatePaneColor(msg) },
	StateProjectRootSetup: func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateProjectRootSetup(msg) },
	StatePekyDialog:       func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updatePekyDialog(msg) },
	StateAuthDialog:       func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateAuthDialog(msg) },
	StateRestartNotice:    func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateRestartNotice(msg) },
	StateUpdateDialog:     func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateUpdateDialog(msg) },
	StateUpdateProgress:   func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateUpdateProgress(msg) },
	StateUpdateRestart:    func(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m.updateUpdateRestart(msg) },
}

type updateHandler func(*Model, tea.Msg) (tea.Model, tea.Cmd)

var updateHandlers = map[reflect.Type]updateHandler{
	reflect.TypeOf(tea.WindowSizeMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		m.applyWindowSize(msg.(tea.WindowSizeMsg))
		return m, nil
	},
	reflect.TypeOf(updateCheckMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleUpdateCheck(msg.(updateCheckMsg))
	},
	reflect.TypeOf(updateCheckResultMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleUpdateCheckResult(msg.(updateCheckResultMsg))
	},
	reflect.TypeOf(updateTickMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleUpdateTick()
	},
	reflect.TypeOf(updateProgressMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleUpdateProgress(msg.(updateProgressMsg))
	},
	reflect.TypeOf(updateInstallResultMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleUpdateInstallResult(msg.(updateInstallResultMsg))
	},
	reflect.TypeOf(updateRestartMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleUpdateRestart(msg.(updateRestartMsg))
	},
	reflect.TypeOf(cursorShapeFlushMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleCursorShapeFlush(msg.(cursorShapeFlushMsg))
	},
	reflect.TypeOf(mouseSendPumpResultMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleMouseSendPumpResult(msg.(mouseSendPumpResultMsg))
	},
	reflect.TypeOf(mouseSendWheelFlushMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleMouseSendWheelFlush(msg.(mouseSendWheelFlushMsg))
	},
	reflect.TypeOf(terminalScrollPumpResultMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleTerminalScrollPumpResult(msg.(terminalScrollPumpResultMsg))
	},
	reflect.TypeOf(terminalScrollWheelFlushMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleTerminalScrollWheelFlush(msg.(terminalScrollWheelFlushMsg))
	},
	reflect.TypeOf(resizeDragFlushMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleResizeDragFlush(msg.(resizeDragFlushMsg))
	},
	reflect.TypeOf(refreshTickMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleRefreshTick(msg.(refreshTickMsg))
	},
	reflect.TypeOf(pekySpinnerTickMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePekySpinnerTick()
	},
	reflect.TypeOf(paneTopbarSpinnerTickMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePaneTopbarSpinnerTick()
	},
	reflect.TypeOf(pekyPromptClearMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePekyPromptClear(msg.(pekyPromptClearMsg))
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
	reflect.TypeOf(daemonReconnectMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleDaemonReconnect(msg.(daemonReconnectMsg))
	},
	reflect.TypeOf(daemonStopMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleDaemonStop(msg.(daemonStopMsg))
	},
	reflect.TypeOf(PaneClosedMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePaneClosed(msg.(PaneClosedMsg))
	},
	reflect.TypeOf(paneCleanupMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePaneCleanup(msg.(paneCleanupMsg))
	},
	reflect.TypeOf(quickReplySendMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleQuickReplySend(msg.(quickReplySendMsg))
	},
	reflect.TypeOf(pekyResultMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handlePekyResult(msg.(pekyResultMsg))
	},
	reflect.TypeOf(authDoneMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleAuthDone(msg.(authDoneMsg))
	},
	reflect.TypeOf(authCopilotDeviceMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleCopilotDevice(msg.(authCopilotDeviceMsg))
	},
	reflect.TypeOf(authCallbackMsg{}): func(m *Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.handleAuthCallback(msg.(authCallbackMsg))
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
	case StatePerformanceMenu:
		var cmd tea.Cmd
		m.perfMenu, cmd = m.perfMenu.Update(msg)
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
	m.setPerformanceMenuSize()
	m.setDebugMenuSize()
	m.setQuickReplySize()
	m.setPekyDialogSize()
}

func (m *Model) handleRefreshTick(msg refreshTickMsg) tea.Cmd {
	if m.daemonDisconnected {
		cmds := []tea.Cmd{tickCmd(m.settings.RefreshInterval)}
		if cmd := m.scheduleDaemonReconnect(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return tea.Batch(cmds...)
	}
	if m.refreshInFlight == 0 {
		logging.LogEvery(
			context.Background(),
			"tui.refresh.start",
			2*time.Second,
			slog.LevelDebug,
			"tui: refresh tick start",
			slog.Uint64("next_seq", m.refreshSeq+1),
		)
		return tea.Batch(m.startRefreshCmd(), tickCmd(m.settings.RefreshInterval))
	}
	logging.LogEvery(
		context.Background(),
		"tui.refresh.skip",
		2*time.Second,
		slog.LevelDebug,
		"tui: refresh tick skipped",
		slog.Int("in_flight", m.refreshInFlight),
		slog.Uint64("seq", m.refreshSeq),
	)
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
	logging.LogEvery(
		context.Background(),
		"tui.snapshot.recv",
		2*time.Second,
		slog.LevelDebug,
		"tui: snapshot recv",
		slog.Uint64("seq", msg.Result.RefreshSeq),
		slog.Any("err", msg.Result.Err),
		slog.Int("in_flight", m.refreshInFlight),
	)
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
	if cmd := m.handleDaemonDisconnect(msg.Result.Err); cmd != nil {
		return cmd
	}
	m.setToast("Refresh failed: "+msg.Result.Err.Error(), toastError)
	return m.appendQueuedRefresh(m.refreshPaneViewsCmd())
}

func (m *Model) applySnapshotState(msg dashboardSnapshotMsg) {
	m.lastAppliedSeq = msg.Result.RefreshSeq
	if msg.Result.Warning != "" {
		m.setToast("Dashboard config: "+msg.Result.Warning, toastWarning)
	}
	prevData := m.data
	m.data = msg.Result.Data
	m.syncLayoutEngines()
	m.reconcilePaneInputDisabled()
	m.settings = msg.Result.Settings
	m.config = msg.Result.RawConfig
	if msg.Result.Keymap != nil {
		m.keys = msg.Result.Keymap
	}
	m.mergePanePreviews(prevData)
	m.pruneOfflineScroll()
	if msg.Result.Version == m.selectionVersion {
		m.applySelection(msg.Result.Resolved)
	} else {
		m.applySelection(resolveSelectionForTab(m.tab, m.data.Projects, m.selection))
	}
	m.updatePaneAgentUnread()
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
	if spinnerCmd := m.maybeStartPaneTopbarSpinner(); spinnerCmd != nil {
		cmds = append(cmds, spinnerCmd)
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
	logging.LogEvery(
		context.Background(),
		"tui.event",
		2*time.Second,
		slog.LevelDebug,
		"tui: events batch",
		slog.Int("batch", len(events)),
		slog.Int("panes", len(paneIDs)),
		slog.Bool("refresh", refresh),
	)
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
		case sessiond.EventPaneMetaChanged:
			refresh = true
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
	var disconnectCmd tea.Cmd
	logging.LogEvery(
		context.Background(),
		"tui.paneviews.recv",
		2*time.Second,
		slog.LevelDebug,
		"tui: paneViews recv",
		slog.Int("count", len(msg.Views)),
		slog.Any("err", msg.Err),
		slog.Int("in_flight", m.paneViewInFlight),
		slog.Int("pending", len(m.paneViewQueuedIDs)),
	)
	if msg.Err != nil {
		disconnectCmd = m.handleDaemonDisconnect(msg.Err)
	}
	if disconnectCmd == nil && msg.Err != nil && len(msg.Views) == 0 && !errors.Is(msg.Err, context.DeadlineExceeded) && !errors.Is(msg.Err, context.Canceled) {
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
	if disconnectCmd != nil {
		if cmd == nil {
			return disconnectCmd
		}
		return tea.Batch(cmd, disconnectCmd)
	}
	return cmd
}

func (m *Model) ensurePaneViewMaps() {
	if m.paneViews == nil {
		m.paneViews = make(map[paneViewKey]paneViewEntry)
	}
	if m.paneHasMouse == nil {
		m.paneHasMouse = make(map[string]bool)
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
	if !view.NotModified && !view.Frame.Empty() {
		m.paneViews[key] = paneViewEntry{
			frame:    view.Frame,
			rendered: make(map[bool]string, 2),
		}
	}
	if view.PaneID != "" {
		m.paneHasMouse[view.PaneID] = view.HasMouse
		m.paneMouseMotion[view.PaneID] = view.AllowMotion
	}
	if perfDebugEnabled() && view.PaneID != "" && !view.Frame.Empty() {
		if m.paneViewFirst == nil {
			m.paneViewFirst = make(map[string]struct{})
		}
		if _, ok := m.paneViewFirst[view.PaneID]; !ok {
			m.paneViewFirst[view.PaneID] = struct{}{}
			logPerfEvery("tui.paneview.first."+view.PaneID, 0, "tui: pane view first pane=%s cols=%d rows=%d", view.PaneID, view.Cols, view.Rows)
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
		sel := m.selection
		if projectID != "" {
			sel.ProjectID = projectID
		}
		sel.Session = msg.Name
		sel.Pane = ""
		m.applySelection(sel)
		m.selectionVersion++
	} else {
		m.setToast("Session started", toastSuccess)
	}
	return m.requestRefreshCmd()
}
