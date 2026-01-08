package app

import (
	"context"
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

const (
	daemonReconnectMinDelay = 500 * time.Millisecond
	daemonReconnectMaxDelay = 10 * time.Second
	daemonReconnectTimeout  = daemonRestartTimeout
	reconnectToastInterval  = 5 * time.Second
)

func (m *Model) handleDaemonDisconnect(err error) tea.Cmd {
	if m == nil || !sessiond.IsConnectionError(err) {
		return nil
	}
	if !m.daemonDisconnected {
		m.daemonDisconnected = true
		m.reconnectToastAt = time.Now()
		m.setToast("Daemon disconnected. Reconnecting...", toastWarning)
	}
	if m.client != nil {
		_ = m.client.Close()
	}
	if m.paneViewClient != nil && m.paneViewClient != m.client {
		_ = m.paneViewClient.Close()
	}
	m.client = nil
	m.paneViewClient = nil
	return m.scheduleDaemonReconnect()
}

func (m *Model) scheduleDaemonReconnect() tea.Cmd {
	if m == nil || m.reconnectInFlight {
		return nil
	}
	delay := nextDaemonReconnectDelay(m.reconnectBackoff)
	m.reconnectBackoff = delay
	m.reconnectInFlight = true
	version := strings.TrimSpace(m.daemonVersion)
	if version == "" && m.client != nil {
		version = strings.TrimSpace(m.client.Version())
		m.daemonVersion = version
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return reconnectDaemon(version)
	})
}

func nextDaemonReconnectDelay(current time.Duration) time.Duration {
	if current <= 0 {
		return daemonReconnectMinDelay
	}
	next := current * 2
	if next > daemonReconnectMaxDelay {
		next = daemonReconnectMaxDelay
	}
	return next
}

func reconnectDaemon(version string) daemonReconnectMsg {
	if strings.TrimSpace(version) == "" {
		return daemonReconnectMsg{Err: errors.New("daemon version unavailable")}
	}
	ctx, cancel := context.WithTimeout(context.Background(), daemonReconnectTimeout)
	defer cancel()
	client, err := sessiond.ConnectDefault(ctx, version)
	if err != nil {
		return daemonReconnectMsg{Err: err}
	}
	paneClient, err := client.Clone(ctx)
	if err != nil {
		_ = client.Close()
		return daemonReconnectMsg{Err: err}
	}
	return daemonReconnectMsg{Client: client, PaneViewClient: paneClient}
}

func (m *Model) handleDaemonReconnect(msg daemonReconnectMsg) tea.Cmd {
	wasDisconnected := m.daemonDisconnected
	m.reconnectInFlight = false
	if msg.Err != nil || msg.Client == nil {
		if m.daemonDisconnected && time.Since(m.reconnectToastAt) >= reconnectToastInterval {
			m.reconnectToastAt = time.Now()
			if msg.Err != nil {
				m.setToast("Reconnect failed: "+msg.Err.Error(), toastWarning)
			} else {
				m.setToast("Reconnect failed: daemon client unavailable", toastWarning)
			}
		}
		return m.scheduleDaemonReconnect()
	}
	m.client = msg.Client
	m.paneViewClient = msg.PaneViewClient
	m.daemonDisconnected = false
	if wasDisconnected {
		m.restartNoticePending = true
	}
	m.reconnectBackoff = 0
	m.reconnectToastAt = time.Time{}
	m.paneViewSeq = nil
	m.paneViewInFlightByPane = nil
	m.paneViewInFlight = 0
	m.paneViewPumpScheduled = false
	m.paneViewPumpBackoff = 0
	m.setToast("Daemon reconnected", toastSuccess)
	cmds := []tea.Cmd{waitDaemonEvent(m.client)}
	if cmd := m.requestRefreshCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := m.schedulePaneViewPump("reconnect", 0); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}
