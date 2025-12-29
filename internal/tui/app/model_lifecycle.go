package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

const (
	defaultSnapshotTimeout = 2 * time.Second
	snapshotBudgetPadding  = 200 * time.Millisecond
)

func snapshotBudget(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 0
	}
	if timeout <= snapshotBudgetPadding {
		return timeout
	}
	return timeout - snapshotBudgetPadding
}

// SetAutoStart queues a session to start when the TUI launches.
func (m *Model) SetAutoStart(spec AutoStartSpec) {
	m.autoStart = &spec
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.startRefreshCmd(), tickCmd(m.settings.RefreshInterval)}
	if m.client != nil {
		cmds = append(cmds, waitDaemonEvent(m.client))
	}
	if m.autoStart != nil {
		cmds = append(cmds, m.startSessionNative(m.autoStart.Session, m.autoStart.Path, m.autoStart.Layout, m.autoStart.Focus))
	}
	return tea.Batch(cmds...)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func waitDaemonEvent(client *sessiond.Client) tea.Cmd {
	if client == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-client.Events()
		if !ok {
			return nil
		}
		return daemonEventMsg{Event: event}
	}
}

func (m *Model) selectionRefreshCmd() tea.Cmd {
	version := m.selectionVersion
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return selectionRefreshMsg{Version: version}
	})
}

func (m *Model) beginRefresh() uint64 {
	m.refreshInFlight++
	m.refreshing = true
	m.refreshSeq++
	return m.refreshSeq
}

func (m *Model) endRefresh() {
	if m.refreshInFlight > 0 {
		m.refreshInFlight--
	}
	m.refreshing = m.refreshInFlight > 0
}

func (m Model) refreshCmd(seq uint64) tea.Cmd {
	selection := m.selection
	currentTab := m.tab
	configPath := m.configPath
	version := m.selectionVersion
	currentSettings := m.settings
	currentKeys := m.keys
	client := m.client
	return func() tea.Msg {
		cfg, err := loadConfig(configPath)
		warning := ""
		if err != nil {
			warning = "config: " + err.Error()
			cfg = &layout.Config{}
		}
		settings, err := defaultDashboardConfig(cfg.Dashboard)
		if err != nil {
			if warning != "" {
				warning += "; "
			}
			warning += "dashboard: " + err.Error()
			settings = currentSettings
		}
		keys, err := buildDashboardKeyMap(cfg.Dashboard.Keymap)
		if err != nil {
			if warning != "" {
				warning += "; "
			}
			warning += "keymap: " + err.Error()
			keys = currentKeys
		}
		var sessions []native.SessionSnapshot
		if client != nil {
			previewLines := settings.PreviewLines
			if dashboard := dashboardPreviewLines(settings); dashboard > previewLines {
				previewLines = dashboard
			}
			timeout := defaultSnapshotTimeout
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			sessions, _, err = client.Snapshot(ctx, previewLines, snapshotBudget(timeout))
			if err != nil {
				result := buildDashboardData(dashboardSnapshotInput{
					Selection:  selection,
					Tab:        currentTab,
					Version:    version,
					RefreshSeq: seq,
					Config:     cfg,
					Settings:   settings,
					Sessions:   nil,
				})
				result.Keymap = keys
				result.Warning = warning
				result.Err = err
				return dashboardSnapshotMsg{Result: result}
			}
		}
		result := buildDashboardData(dashboardSnapshotInput{
			Selection:  selection,
			Tab:        currentTab,
			Version:    version,
			RefreshSeq: seq,
			Config:     cfg,
			Settings:   settings,
			Sessions:   sessions,
		})
		result.Keymap = keys
		result.Warning = warning
		return dashboardSnapshotMsg{Result: result}
	}
}

func (m *Model) startRefreshCmd() tea.Cmd {
	if m == nil {
		return nil
	}
	seq := m.beginRefresh()
	return m.refreshCmd(seq)
}

func (m *Model) requestRefreshCmd() tea.Cmd {
	if m == nil {
		return nil
	}
	if m.refreshInFlight > 0 {
		m.refreshQueued = true
		return nil
	}
	return m.startRefreshCmd()
}
