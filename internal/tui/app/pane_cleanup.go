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
	Session   string
	Restarted bool
	Closed    int
	Added     int
	Failed    int
	Err       string
	Noop      string
}

func (m *Model) cleanupDeadPanes() tea.Cmd {
	session := m.selectedSession()
	if session == nil {
		m.setToast("No session selected", toastWarning)
		return nil
	}
	if m.client == nil {
		m.setToast("Pane cleanup failed: session client unavailable", toastError)
		return nil
	}
	dead, live := splitDeadPanes(session.Panes)
	if len(dead) == 0 {
		return NewInfoCmd("No dead/offline panes")
	}
	if len(live) == 0 {
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
				return paneCleanupMsg{Session: session.Name, Err: err.Error()}
			}
			return paneCleanupMsg{Session: resp.Name, Restarted: true}
		}
	}
	anchor := selectCleanupAnchor(live)
	if anchor == nil {
		return NewWarningCmd("No live pane available for cleanup")
	}
	target := len(dead)
	vertical := autoSplitVertical(anchor.Width, anchor.Height)
	sessionName := session.Name
	anchorIndex := anchor.Index
	return func() tea.Msg {
		result := paneCleanupMsg{Session: sessionName}
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
		for i := 0; i < target; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), paneCleanupTimeout)
			_, err := m.client.SplitPane(ctx, sessionName, anchorIndex, vertical, 0)
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
	if msg.Restarted {
		name := strings.TrimSpace(msg.Session)
		if name == "" {
			m.setToast("Restarted session", toastSuccess)
		} else {
			m.setToast("Restarted session "+name, toastSuccess)
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
