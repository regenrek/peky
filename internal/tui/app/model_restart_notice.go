package app

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const restartNoticeFlagFile = ".pp-restart-notice"

func (m *Model) updateRestartNotice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(strings.TrimSpace(msg.String())) {
	case "esc", "q":
		m.setState(StateDashboard)
		return m, nil
	case "enter", "f":
		m.setRestartNoticePending(false)
		m.setState(StateDashboard)
		m.startFreshFromRestart()
		return m, nil
	case "s":
		m.setState(StateDashboard)
		return m, m.focusFirstStalePane()
	}
	return m, nil
}

func (m *Model) restartNoticeFlagPath() string {
	if m == nil {
		return ""
	}
	configPath := strings.TrimSpace(m.configPath)
	if configPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(configPath), restartNoticeFlagFile)
}

func restartNoticeFlagActive(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

func (m *Model) setRestartNoticePending(active bool) {
	if m == nil {
		return
	}
	m.restartNoticePending = active
	path := m.restartNoticeFlagPath()
	if path == "" {
		return
	}
	value := "0\n"
	if active {
		value = "1\n"
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(value), 0o600)
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
