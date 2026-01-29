package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

const paneCleanupTimeout = 4 * time.Second

type paneCleanupMsg struct {
	Sessions  []string
	Restarted int
	Closed    int
	Added     int
	Failed    int
	Err       string
	Noop      string
}

type restartSessionTarget struct {
	name   string
	path   string
	layout string
}

func (m *Model) cleanupDeadPanes() tea.Cmd {
	if m.client == nil {
		m.setToast("Pane cleanup failed: session client unavailable", toastError)
		return nil
	}
	if cmd := m.restartOfflineSessionsIfNeeded(); cmd != nil {
		return cmd
	}
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	return m.cleanupDeadPanesForSession(session)
}

func (m *Model) restartOfflineSessionsIfNeeded() tea.Cmd {
	targets, hasLive := collectOfflineSessions(m.data.Projects)
	if !hasLive && len(targets) > 0 {
		return m.restartOfflineSessionsCmd(targets)
	}
	return nil
}

func (m *Model) cleanupDeadPanesForSession(session *SessionItem) tea.Cmd {
	dead, live := splitDeadPanes(session.Panes)
	if len(dead) == 0 {
		return NewInfoCmd("No dead/offline panes")
	}
	if len(live) == 0 {
		return m.restartSessionCmd(session)
	}
	anchor := selectCleanupAnchor(live)
	if anchor == nil {
		return NewWarningCmd("No live pane available for cleanup")
	}
	return m.cleanupPanesWithAnchor(session, anchor, dead)
}

func (m *Model) restartSessionCmd(session *SessionItem) tea.Cmd {
	path := strings.TrimSpace(session.Path)
	if path == "" {
		return NewWarningCmd("Session path missing; cannot restart")
	}
	req := sessiond.StartSessionRequest{
		Name:       session.Name,
		Path:       path,
		LayoutName: session.LayoutName,
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), paneCleanupTimeout)
		defer cancel()
		resp, err := m.client.StartSession(ctx, req)
		if err != nil {
			return paneCleanupMsg{Err: err.Error()}
		}
		name := strings.TrimSpace(resp.Name)
		if name == "" {
			name = session.Name
		}
		return paneCleanupMsg{Sessions: []string{name}, Restarted: 1}
	}
}

func (m *Model) cleanupPanesWithAnchor(session *SessionItem, anchor *PaneItem, dead []PaneItem) tea.Cmd {
	vertical := autoSplitVertical(anchor.Width, anchor.Height)
	sessionName := session.Name
	anchorIndex := anchor.Index
	return func() tea.Msg {
		return m.cleanupPanesRun(sessionName, anchorIndex, vertical, dead)
	}
}

func (m *Model) cleanupPanesRun(sessionName, anchorIndex string, vertical bool, dead []PaneItem) paneCleanupMsg {
	result := paneCleanupMsg{}
	for _, pane := range dead {
		ctx, cancel := context.WithTimeout(context.Background(), paneCleanupTimeout)
		err := m.client.ClosePaneByID(ctx, pane.ID)
		cancel()
		if err != nil {
			result.Failed++
			continue
		}
		result.Closed++
	}
	target := len(dead)
	for i := 0; i < target; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), paneCleanupTimeout)
		_, _, err := m.client.SplitPane(ctx, sessionName, anchorIndex, vertical, 0)
		cancel()
		if err != nil {
			result.Failed++
			continue
		}
		result.Added++
	}
	if result.Closed == 0 && result.Added == 0 && result.Failed == 0 {
		result.Noop = "No panes cleaned"
	}
	if result.Failed > 0 {
		result.Err = fmt.Sprintf("%d operations failed", result.Failed)
	}
	return result
}

func collectOfflineSessions(projects []ProjectGroup) ([]restartSessionTarget, bool) {
	var targets []restartSessionTarget
	hasLive := false
	for _, project := range projects {
		for _, session := range project.Sessions {
			dead, live := splitDeadPanes(session.Panes)
			if len(live) > 0 {
				hasLive = true
			}
			if len(dead) > 0 && len(live) == 0 {
				targets = append(targets, restartSessionTarget{
					name:   session.Name,
					path:   session.Path,
					layout: session.LayoutName,
				})
			}
		}
	}
	return targets, hasLive
}

func (m *Model) restartOfflineSessionsCmd(targets []restartSessionTarget) tea.Cmd {
	if len(targets) == 0 {
		return nil
	}
	return func() tea.Msg {
		result := paneCleanupMsg{}
		for _, target := range targets {
			path := strings.TrimSpace(target.path)
			if path == "" {
				result.Failed++
				continue
			}
			req := sessiond.StartSessionRequest{
				Name:       target.name,
				Path:       path,
				LayoutName: target.layout,
			}
			ctx, cancel := context.WithTimeout(context.Background(), paneCleanupTimeout)
			resp, err := m.client.StartSession(ctx, req)
			cancel()
			if err != nil {
				result.Failed++
				continue
			}
			name := strings.TrimSpace(resp.Name)
			if name == "" {
				name = target.name
			}
			result.Sessions = append(result.Sessions, name)
		}
		result.Restarted = len(result.Sessions)
		if result.Restarted == 0 && result.Failed > 0 {
			result.Err = fmt.Sprintf("%d session restart(s) failed", result.Failed)
		}
		if result.Restarted == 0 && result.Failed == 0 {
			result.Noop = "No dead/offline panes"
		}
		return result
	}
}

func splitDeadPanes(panes []PaneItem) (dead []PaneItem, live []PaneItem) {
	for _, pane := range panes {
		if pane.Dead || pane.Disconnected {
			dead = append(dead, pane)
			continue
		}
		live = append(live, pane)
	}
	return dead, live
}

func selectCleanupAnchor(panes []PaneItem) *PaneItem {
	for i := range panes {
		if panes[i].Active {
			return &panes[i]
		}
	}
	if len(panes) > 0 {
		return &panes[0]
	}
	return nil
}

func (m *Model) handlePaneCleanup(msg paneCleanupMsg) tea.Cmd {
	if msg.Noop != "" {
		m.setToast(msg.Noop, toastInfo)
		return nil
	}
	if msg.Err != "" {
		m.setToast("Pane cleanup failed: "+msg.Err, toastError)
		return m.requestRefreshCmd()
	}
	if msg.Restarted > 0 {
		label := "Restarted session"
		if msg.Restarted > 1 {
			label = fmt.Sprintf("Restarted %d sessions", msg.Restarted)
		} else if len(msg.Sessions) == 1 && strings.TrimSpace(msg.Sessions[0]) != "" {
			label = "Restarted session " + strings.TrimSpace(msg.Sessions[0])
		}
		if msg.Failed > 0 {
			label = fmt.Sprintf("%s (%d failed)", label, msg.Failed)
			m.setToast(label, toastWarning)
		} else {
			m.setToast(label, toastSuccess)
		}
		return m.requestRefreshCmd()
	}
	if msg.Added > 0 {
		label := fmt.Sprintf("Recreated %d pane", msg.Added)
		if msg.Added != 1 {
			label = fmt.Sprintf("Recreated %d panes", msg.Added)
		}
		if msg.Failed > 0 {
			label = fmt.Sprintf("%s (%d failed)", label, msg.Failed)
			m.setToast(label, toastWarning)
		} else {
			m.setToast(label, toastSuccess)
		}
		return m.requestRefreshCmd()
	}
	if msg.Closed > 0 {
		label := fmt.Sprintf("Closed %d pane", msg.Closed)
		if msg.Closed != 1 {
			label = fmt.Sprintf("Closed %d panes", msg.Closed)
		}
		m.setToast(label, toastWarning)
		return m.requestRefreshCmd()
	}
	if msg.Failed > 0 {
		m.setToast(fmt.Sprintf("Pane cleanup failed (%d errors)", msg.Failed), toastError)
		return m.requestRefreshCmd()
	}
	return nil
}
