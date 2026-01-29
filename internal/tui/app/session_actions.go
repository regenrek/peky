package app

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) killProjectSessions() tea.Cmd {
	name := strings.TrimSpace(m.confirmClose)
	projectID := strings.TrimSpace(m.confirmCloseID)
	m.confirmClose = ""
	m.confirmCloseID = ""
	m.setState(StateDashboard)
	if projectID == "" {
		return nil
	}
	project := findProjectByID(m.data.Projects, projectID)
	if project == nil {
		m.setToast("Project not found", toastWarning)
		return nil
	}
	if name == "" {
		name = project.Name
	}
	var running []SessionItem
	for _, s := range project.Sessions {
		if s.Status != StatusStopped {
			running = append(running, s)
		}
	}
	if len(running) == 0 {
		m.setToast("No running sessions to kill", toastInfo)
		return nil
	}
	var failed []string
	for _, s := range running {
		if m.client == nil {
			failed = append(failed, s.Name)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := m.client.KillSession(ctx, s.Name); err != nil {
			failed = append(failed, s.Name)
		}
		cancel()
	}
	if len(failed) > 0 {
		m.setToast("Kill failed: "+strings.Join(failed, ", "), toastError)
		return m.requestRefreshCmd()
	}
	m.setToast("Killed sessions for "+name, toastSuccess)
	return m.requestRefreshCmd()
}

func (m *Model) killAllProjectSessions() tea.Cmd {
	m.setState(StateDashboard)
	if len(m.data.Projects) == 0 {
		m.setToast("No running sessions to kill", toastInfo)
		return nil
	}
	var running []SessionItem
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			if session.Status != StatusStopped {
				running = append(running, session)
			}
		}
	}
	if len(running) == 0 {
		m.setToast("No running sessions to kill", toastInfo)
		return nil
	}
	var failed []string
	for _, s := range running {
		if m.client == nil {
			failed = append(failed, s.Name)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := m.client.KillSession(ctx, s.Name); err != nil {
			failed = append(failed, s.Name)
		}
		cancel()
	}
	if len(failed) > 0 {
		m.setToast("Kill failed: "+strings.Join(failed, ", "), toastError)
		return m.requestRefreshCmd()
	}
	m.setToast("Killed all running sessions", toastSuccess)
	return m.requestRefreshCmd()
}

func runningPaneCount(projects []ProjectGroup) int {
	count := 0
	for _, project := range projects {
		for _, session := range project.Sessions {
			for _, pane := range session.Panes {
				if !pane.Dead && !pane.Disconnected {
					count++
				}
			}
		}
	}
	return count
}

func (m *Model) updateConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.confirmSession != "" {
			if m.client == nil {
				m.setToast("Kill failed: session client unavailable", toastError)
				m.setState(StateDashboard)
				return m, nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := m.client.KillSession(ctx, m.confirmSession); err != nil {
				m.setToast("Kill failed: "+err.Error(), toastError)
				m.setState(StateDashboard)
				return m, nil
			}
			m.setToast("Killed session "+m.confirmSession, toastSuccess)
			m.confirmSession = ""
			m.confirmProject = ""
			m.setState(StateDashboard)
			return m, m.requestRefreshCmd()
		}
		m.setState(StateDashboard)
		return m, nil
	case "n", "esc":
		m.confirmSession = ""
		m.confirmProject = ""
		m.setState(StateDashboard)
		return m, nil
	}
	return m, nil
}
