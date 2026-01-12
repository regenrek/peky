package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handlePaneTopbarClick(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil || !m.settings.PaneTopbar.Enabled {
		return nil, false
	}
	if m.state != StateDashboard || m.tab != TabProject {
		return nil, false
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil, false
	}
	hit, ok := m.hitTestPane(msg.X, msg.Y)
	if !ok || hit.Topbar.Empty() || !hit.Topbar.Contains(msg.X, msg.Y) {
		return nil, false
	}
	m.applySelection(selectionFromMouse(hit.Selection))
	m.selectionVersion++

	pane := m.paneByID(hit.PaneID)
	if pane == nil {
		return nil, true
	}
	m.openPekyDialog("Pane details", paneTopbarDialogBody(*pane), "esc close", false)
	return nil, true
}

func paneTopbarDialogBody(pane PaneItem) string {
	lines := []string{
		fmt.Sprintf("Pane:   %s", strings.TrimSpace(pane.Index)),
		fmt.Sprintf("ID:     %s", strings.TrimSpace(pane.ID)),
		fmt.Sprintf("CWD:    %s", strings.TrimSpace(pane.Cwd)),
	}
	if strings.TrimSpace(pane.GitRoot) != "" {
		lines = append(lines, fmt.Sprintf("Git:    %s", strings.TrimSpace(pane.GitRoot)))
	}
	if strings.TrimSpace(pane.GitBranch) != "" {
		attrs := []string{}
		if pane.GitDirty {
			attrs = append(attrs, "dirty")
		}
		if pane.GitWorktree {
			attrs = append(attrs, "worktree")
		}
		suffix := ""
		if len(attrs) > 0 {
			suffix = " (" + strings.Join(attrs, ", ") + ")"
		}
		lines = append(lines, fmt.Sprintf("Head:   %s%s", strings.TrimSpace(pane.GitBranch), suffix))
	}
	if strings.TrimSpace(pane.AgentTool) != "" {
		state := strings.TrimSpace(pane.AgentState)
		if state == "" {
			state = "idle"
		}
		unread := "no"
		if pane.AgentUnread {
			unread = "yes"
		}
		lines = append(lines,
			fmt.Sprintf("Agent:  %s (%s)", strings.TrimSpace(pane.AgentTool), state),
			fmt.Sprintf("Unread: %s", unread),
		)
		if !pane.AgentUpdated.IsZero() {
			lines = append(lines, fmt.Sprintf("Agent updated: %s", pane.AgentUpdated.Format("2006-01-02 15:04:05")))
		}
	}
	return strings.Join(lines, "\n")
}
