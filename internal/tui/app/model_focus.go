package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/logging"
)

func (m *Model) queueFocusSync(prev, next selectionState) {
	if m == nil || prev == next {
		return
	}
	m.focusSelection = next
	m.focusPending = true
}

func (m *Model) appendFocusCmd(cmd tea.Cmd) tea.Cmd {
	if m == nil {
		return cmd
	}
	focusCmd := m.consumeFocusSyncCmd()
	if focusCmd == nil {
		return cmd
	}
	if cmd == nil {
		return focusCmd
	}
	return tea.Batch(cmd, focusCmd)
}

func (m *Model) consumeFocusSyncCmd() tea.Cmd {
	if m == nil || !m.focusPending {
		return nil
	}
	m.focusPending = false
	if m.client == nil {
		return nil
	}
	return m.focusSelectionCmd(m.focusSelection)
}

func (m *Model) focusSelectionCmd(sel selectionState) tea.Cmd {
	if m == nil {
		return nil
	}
	client := m.client
	if client == nil {
		return nil
	}
	if pane := m.paneForSelection(sel); pane != nil && strings.TrimSpace(pane.ID) != "" {
		paneID := pane.ID
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
			defer cancel()
			if err := client.FocusPane(ctx, paneID); err != nil {
				logging.LogEvery(
					context.Background(),
					"tui.focus.pane",
					2*time.Second,
					slog.LevelWarn,
					"tui: focus pane failed",
					slog.String("pane_id", paneID),
					slog.Any("err", err),
				)
			}
			return nil
		}
	}
	session := strings.TrimSpace(sel.Session)
	if session == "" {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if err := client.FocusSession(ctx, session); err != nil {
			logging.LogEvery(
				context.Background(),
				"tui.focus.session",
				2*time.Second,
				slog.LevelWarn,
				"tui: focus session failed",
				slog.String("session", session),
				slog.Any("err", err),
			)
		}
		return nil
	}
}

func (m *Model) paneForSelection(sel selectionState) *PaneItem {
	if m == nil {
		return nil
	}
	sessionName := strings.TrimSpace(sel.Session)
	paneIndex := strings.TrimSpace(sel.Pane)
	if sessionName == "" || paneIndex == "" {
		return nil
	}
	session := findSessionByName(m.data.Projects, sessionName)
	if session == nil {
		return nil
	}
	for i := range session.Panes {
		if session.Panes[i].Index == paneIndex {
			return &session.Panes[i]
		}
	}
	return nil
}
