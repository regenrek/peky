package app

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

const daemonRestartTimeout = 15 * time.Second

func (m *Model) openRestartConfirm() {
	m.setState(StateConfirmRestart)
}

func (m *Model) updateConfirmRestart(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.setState(StateDashboard)
		m.setToast("Restarting daemon...", toastInfo)
		return m, m.restartDaemonCmd()
	case "n", "esc":
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
}

func (m *Model) restartDaemonCmd() tea.Cmd {
	client := m.client
	paneViewClient := m.paneViewClient
	version := ""
	if client != nil {
		version = client.Version()
	}
	return func() tea.Msg {
		if version == "" {
			return daemonRestartMsg{Err: errors.New("daemon version unavailable")}
		}
		if client != nil {
			_ = client.Close()
		}
		if paneViewClient != nil && paneViewClient != client {
			_ = paneViewClient.Close()
		}

		ctx, cancel := context.WithTimeout(context.Background(), daemonRestartTimeout)
		defer cancel()
		if err := sessiond.RestartDaemon(ctx, version); err != nil {
			return daemonRestartMsg{Err: err}
		}

		connectCtx, connectCancel := context.WithTimeout(context.Background(), daemonRestartTimeout)
		defer connectCancel()
		newClient, err := sessiond.ConnectDefault(connectCtx, version)
		if err != nil {
			return daemonRestartMsg{Err: err}
		}
		paneClient, err := newClient.Clone(connectCtx)
		if err != nil {
			_ = newClient.Close()
			return daemonRestartMsg{Err: err}
		}
		return daemonRestartMsg{Client: newClient, PaneViewClient: paneClient}
	}
}

func (m *Model) handleDaemonRestart(msg daemonRestartMsg) tea.Cmd {
	if msg.Err != nil {
		m.client = nil
		m.paneViewClient = nil
		m.paneViews = nil
		m.paneViewQueuedIDs = nil
		m.paneViewQueued = false
		m.paneViewInFlight = false
		m.setToast("Restart failed: "+msg.Err.Error(), toastError)
		return nil
	}
	if msg.Client == nil {
		m.client = nil
		m.paneViewClient = nil
		m.paneViews = nil
		m.paneViewQueuedIDs = nil
		m.paneViewQueued = false
		m.paneViewInFlight = false
		m.setToast("Restart failed: daemon client unavailable", toastError)
		return nil
	}
	m.client = msg.Client
	m.paneViewClient = msg.PaneViewClient
	m.paneViews = nil
	m.paneViewQueuedIDs = nil
	m.paneViewQueued = false
	m.paneViewInFlight = false
	if m.paneMouseMotion == nil {
		m.paneMouseMotion = make(map[string]bool)
	} else {
		for key := range m.paneMouseMotion {
			delete(m.paneMouseMotion, key)
		}
	}
	m.setToast("Daemon restarted", toastSuccess)
	return tea.Batch(waitDaemonEvent(m.client), m.requestRefreshCmd())
}
