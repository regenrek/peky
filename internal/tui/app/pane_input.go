package app

import "strings"

func (m *Model) isPaneInputDisabled(paneID string) bool {
	if m == nil || paneID == "" || m.paneInputDisabled == nil {
		return false
	}
	_, ok := m.paneInputDisabled[paneID]
	return ok
}

func (m *Model) markPaneInputDisabled(paneID string) {
	if m == nil || paneID == "" {
		return
	}
	if m.paneInputDisabled == nil {
		m.paneInputDisabled = make(map[string]struct{})
	}
	m.paneInputDisabled[paneID] = struct{}{}
}

func (m *Model) reconcilePaneInputDisabled() {
	if m == nil || len(m.paneInputDisabled) == 0 {
		return
	}
	deadPanes := make(map[string]struct{})
	for projIdx := range m.data.Projects {
		project := &m.data.Projects[projIdx]
		for sessIdx := range project.Sessions {
			session := &project.Sessions[sessIdx]
			for paneIdx := range session.Panes {
				pane := &session.Panes[paneIdx]
				if pane.Dead || pane.Disconnected {
					deadPanes[pane.ID] = struct{}{}
				}
			}
		}
	}
	for paneID := range m.paneInputDisabled {
		if _, ok := deadPanes[paneID]; !ok {
			delete(m.paneInputDisabled, paneID)
		}
	}
}

func (m *Model) paneByID(paneID string) *PaneItem {
	if m == nil || paneID == "" {
		return nil
	}
	for projIdx := range m.data.Projects {
		project := &m.data.Projects[projIdx]
		for sessIdx := range project.Sessions {
			session := &project.Sessions[sessIdx]
			for paneIdx := range session.Panes {
				if session.Panes[paneIdx].ID == paneID {
					return &session.Panes[paneIdx]
				}
			}
		}
	}
	return nil
}

func (m *Model) selectionForPaneID(paneID string) (selectionState, bool) {
	if m == nil || strings.TrimSpace(paneID) == "" {
		return selectionState{}, false
	}
	for projIdx := range m.data.Projects {
		project := &m.data.Projects[projIdx]
		for sessIdx := range project.Sessions {
			session := &project.Sessions[sessIdx]
			for paneIdx := range session.Panes {
				if session.Panes[paneIdx].ID == paneID {
					return selectionState{
						ProjectID: project.ID,
						Session:   session.Name,
						Pane:      session.Panes[paneIdx].Index,
					}, true
				}
			}
		}
	}
	return selectionState{}, false
}
