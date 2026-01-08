package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateRestartNotice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(strings.TrimSpace(msg.String())) {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "enter", "f":
		m.restartNoticePending = false
		m.setState(StateDashboard)
		m.startFreshFromRestart()
		return m, nil
	case "s":
		m.restartNoticePending = false
		m.setState(StateDashboard)
		return m, m.focusFirstStalePane()
	}
	return m, nil
}

func (m *Model) startFreshFromRestart() {
	if m == nil {
		return
	}
	if len(m.data.Projects) == 0 {
		m.openProjectPicker()
		return
	}
	if m.tab == TabDashboard {
		m.selectTab(1)
	}
	m.openLayoutPicker()
}

func (m *Model) focusFirstStalePane() tea.Cmd {
	if m == nil {
		return nil
	}
	type match struct {
		projectID string
		session   string
		pane      string
		paneID    string
		offline   bool
	}
	var best *match
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			for _, pane := range session.Panes {
				if !pane.Disconnected && !pane.Dead {
					continue
				}
				candidate := match{
					projectID: project.ID,
					session:   session.Name,
					pane:      pane.Index,
					paneID:    pane.ID,
					offline:   pane.Disconnected,
				}
				if best == nil {
					best = &candidate
					continue
				}
				// Prefer disconnected panes (have usable snapshot scrolling).
				if !best.offline && candidate.offline {
					best = &candidate
				}
			}
		}
	}
	if best == nil {
		m.setToast("No stale panes found", toastInfo)
		return nil
	}
	m.tab = TabProject
	m.applySelection(selectionState{
		ProjectID: best.projectID,
		Session:   best.session,
		Pane:      best.pane,
	})
	m.selectionVersion++
	if strings.TrimSpace(best.paneID) != "" {
		if pane := m.paneByID(best.paneID); pane != nil && pane.Disconnected {
			m.toggleOfflineScroll(pane.ID)
			m.setOfflineScrollOffset(*pane, 0)
		}
	}
	m.setToast("Stale pane selected (scroll to view cached output)", toastInfo)
	return m.selectionRefreshCmd()
}
