package app

import (
	"time"

	"github.com/regenrek/peakypanes/internal/tui/icons"

	tea "github.com/charmbracelet/bubbletea"
)

const paneTopbarSpinnerInterval = 120 * time.Millisecond

type paneAgentSeen struct {
	state     string
	updatedAt time.Time
}

func (m *Model) paneTopbarSpinnerTickCmd() tea.Cmd {
	return tea.Tick(paneTopbarSpinnerInterval, func(time.Time) tea.Msg {
		return paneTopbarSpinnerTickMsg{}
	})
}

func (m *Model) maybeStartPaneTopbarSpinner() tea.Cmd {
	if m == nil || !m.settings.PaneTopbar.Enabled || m.paneTopbarSpinnerOn {
		return nil
	}
	if !m.anyPaneAgentRunning() {
		return nil
	}
	frames := icons.Active().Spinner
	if len(frames) == 0 {
		return nil
	}
	m.paneTopbarSpinnerOn = true
	return m.paneTopbarSpinnerTickCmd()
}

func (m *Model) handlePaneTopbarSpinnerTick() tea.Cmd {
	if m == nil || !m.settings.PaneTopbar.Enabled {
		if m != nil {
			m.paneTopbarSpinnerOn = false
		}
		return nil
	}
	if !m.anyPaneAgentRunning() {
		m.paneTopbarSpinnerOn = false
		return nil
	}
	frames := icons.Active().Spinner
	if len(frames) == 0 {
		m.paneTopbarSpinnerOn = false
		return nil
	}
	m.paneTopbarSpinnerIndex = (m.paneTopbarSpinnerIndex + 1) % len(frames)
	return m.paneTopbarSpinnerTickCmd()
}

func (m *Model) paneTopbarSpinnerFrame() string {
	frames := icons.Active().Spinner
	if len(frames) == 0 {
		return ""
	}
	idx := 0
	if m != nil {
		idx = m.paneTopbarSpinnerIndex
	}
	return frames[idx%len(frames)]
}

func (m *Model) anyPaneAgentRunning() bool {
	if m == nil || m.data.Projects == nil {
		return false
	}
	for i := range m.data.Projects {
		project := &m.data.Projects[i]
		for j := range project.Sessions {
			session := &project.Sessions[j]
			for k := range session.Panes {
				pane := &session.Panes[k]
				if pane.AgentState == "running" {
					return true
				}
			}
		}
	}
	return false
}

func (m *Model) updatePaneAgentUnread() {
	if m == nil {
		return
	}
	if m.paneAgentLast == nil {
		m.paneAgentLast = make(map[string]paneAgentSeen)
	}
	if m.paneAgentUnread == nil {
		m.paneAgentUnread = make(map[string]bool)
	}

	selected := m.selectedPaneID()
	for i := range m.data.Projects {
		project := &m.data.Projects[i]
		for j := range project.Sessions {
			session := &project.Sessions[j]
			for k := range session.Panes {
				pane := &session.Panes[k]
				pane.AgentUnread = m.computePaneAgentUnread(pane, selected)
			}
		}
	}
	m.prunePaneAgentTracking()
}

func (m *Model) computePaneAgentUnread(pane *PaneItem, selectedPaneID string) bool {
	if m == nil || pane == nil || pane.ID == "" || pane.AgentTool == "" || pane.AgentUpdated.IsZero() {
		if pane != nil {
			pane.AgentUnread = false
		}
		return false
	}
	if pane.ID == selectedPaneID {
		m.paneAgentUnread[pane.ID] = false
		m.paneAgentLast[pane.ID] = paneAgentSeen{state: pane.AgentState, updatedAt: pane.AgentUpdated}
		return false
	}

	prev, hasPrev := m.paneAgentLast[pane.ID]
	unread := m.paneAgentUnread[pane.ID]

	if pane.AgentState == "running" {
		unread = false
	} else if !hasPrev {
		unread = true
	} else if pane.AgentUpdated.After(prev.updatedAt) {
		unread = true
	}

	m.paneAgentUnread[pane.ID] = unread
	m.paneAgentLast[pane.ID] = paneAgentSeen{state: pane.AgentState, updatedAt: pane.AgentUpdated}
	return unread
}

func (m *Model) clearPaneAgentUnread(paneID string) {
	if m == nil || paneID == "" || m.paneAgentUnread == nil {
		return
	}
	m.paneAgentUnread[paneID] = false
}

func (m *Model) prunePaneAgentTracking() {
	if m == nil || (m.paneAgentLast == nil && m.paneAgentUnread == nil) {
		return
	}
	live := make(map[string]struct{})
	for i := range m.data.Projects {
		project := &m.data.Projects[i]
		for j := range project.Sessions {
			session := &project.Sessions[j]
			for k := range session.Panes {
				id := session.Panes[k].ID
				if id != "" {
					live[id] = struct{}{}
				}
			}
		}
	}
	for id := range m.paneAgentLast {
		if _, ok := live[id]; !ok {
			delete(m.paneAgentLast, id)
		}
	}
	for id := range m.paneAgentUnread {
		if _, ok := live[id]; !ok {
			delete(m.paneAgentUnread, id)
		}
	}
}
