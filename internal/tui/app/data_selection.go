package app

import "strings"

func resolveSelection(groups []ProjectGroup, desired selectionState) selectionState {
	resolved := selectionState{}
	if len(groups) == 0 {
		return resolved
	}
	project := findProjectByID(groups, desired.ProjectID)
	if project == nil {
		project = &groups[0]
	}
	resolved.ProjectID = project.ID
	if len(project.Sessions) == 0 {
		return resolved
	}
	session := findSession(project, desired.Session)
	if session == nil {
		session = &project.Sessions[0]
	}
	resolved.Session = session.Name
	resolved.Pane = desired.Pane
	return resolved
}

func resolveDashboardSelection(groups []ProjectGroup, desired selectionState) selectionState {
	columns := collectDashboardColumns(groups)
	if len(columns) == 0 {
		return selectionState{}
	}
	if selected := resolveDashboardSelectionFromColumns(columns, desired); selected.ProjectID != "" {
		return selected
	}
	return selectionState{}
}

func resolveDashboardSelectionFromColumns(columns []DashboardProjectColumn, desired selectionState) selectionState {
	if len(columns) == 0 {
		return selectionState{}
	}
	if desired.Session != "" {
		for _, column := range columns {
			if len(column.Panes) == 0 {
				continue
			}
			if idx := dashboardPaneIndex(column.Panes, desired); idx >= 0 {
				pane := column.Panes[idx]
				return selectionState{
					ProjectID: column.ProjectID,
					Session:   pane.SessionName,
					Pane:      pane.Pane.Index,
				}
			}
		}
	}
	if desired.ProjectID != "" {
		for _, column := range columns {
			if column.ProjectID != desired.ProjectID {
				continue
			}
			if len(column.Panes) == 0 {
				return selectionState{ProjectID: column.ProjectID}
			}
			idx := dashboardPaneIndex(column.Panes, desired)
			if idx < 0 {
				idx = 0
			}
			pane := column.Panes[idx]
			return selectionState{
				ProjectID: column.ProjectID,
				Session:   pane.SessionName,
				Pane:      pane.Pane.Index,
			}
		}
	}
	for _, column := range columns {
		if len(column.Panes) == 0 {
			continue
		}
		pane := column.Panes[0]
		return selectionState{
			ProjectID: column.ProjectID,
			Session:   pane.SessionName,
			Pane:      pane.Pane.Index,
		}
	}
	if len(columns) > 0 {
		return selectionState{ProjectID: columns[0].ProjectID}
	}
	return selectionState{}
}

func resolvePaneSelection(desired string, panes []PaneItem) string {
	if desired != "" && paneExists(panes, desired) {
		return desired
	}
	if active := activePaneIndex(panes); active != "" {
		return active
	}
	if len(panes) > 0 {
		return panes[0].Index
	}
	return ""
}

func selectionEmpty(sel selectionState) bool {
	return sel.ProjectID == "" && sel.Session == "" && sel.Pane == ""
}

type focusIndex struct {
	byPaneID  map[string]selectionState
	bySession map[string]selectionState
}

func buildFocusIndex(groups []ProjectGroup) focusIndex {
	if len(groups) == 0 {
		return focusIndex{}
	}
	idx := focusIndex{
		byPaneID:  make(map[string]selectionState),
		bySession: make(map[string]selectionState),
	}
	for gi := range groups {
		group := &groups[gi]
		projectID := group.ID
		for si := range group.Sessions {
			session := &group.Sessions[si]
			sessionName := strings.TrimSpace(session.Name)
			if sessionName == "" {
				continue
			}
			if _, exists := idx.bySession[sessionName]; !exists {
				selected := selectionState{ProjectID: projectID, Session: sessionName}
				if len(session.Panes) > 0 {
					selected.Pane = resolvePaneSelection("", session.Panes)
				}
				idx.bySession[sessionName] = selected
			}
			for pi := range session.Panes {
				pane := &session.Panes[pi]
				paneID := strings.TrimSpace(pane.ID)
				if paneID == "" {
					continue
				}
				if _, exists := idx.byPaneID[paneID]; !exists {
					idx.byPaneID[paneID] = selectionState{
						ProjectID: projectID,
						Session:   sessionName,
						Pane:      pane.Index,
					}
				}
			}
		}
	}
	if len(idx.byPaneID) == 0 {
		idx.byPaneID = nil
	}
	if len(idx.bySession) == 0 {
		idx.bySession = nil
	}
	return idx
}

func selectionFromFocus(index focusIndex, focusedSession, focusedPaneID string) selectionState {
	paneID := strings.TrimSpace(focusedPaneID)
	if paneID != "" && index.byPaneID != nil {
		if sel, ok := index.byPaneID[paneID]; ok {
			return sel
		}
	}

	sessionName := strings.TrimSpace(focusedSession)
	if sessionName == "" || index.bySession == nil {
		return selectionState{}
	}
	if sel, ok := index.bySession[sessionName]; ok {
		return sel
	}
	return selectionState{}
}

func findProjectByID(groups []ProjectGroup, id string) *ProjectGroup {
	for i := range groups {
		if groups[i].ID == id {
			return &groups[i]
		}
	}
	return nil
}

func findSession(group *ProjectGroup, name string) *SessionItem {
	if group == nil {
		return nil
	}
	for i := range group.Sessions {
		if group.Sessions[i].Name == name {
			return &group.Sessions[i]
		}
	}
	return nil
}

func findSessionByName(groups []ProjectGroup, name string) *SessionItem {
	for gi := range groups {
		for si := range groups[gi].Sessions {
			if groups[gi].Sessions[si].Name == name {
				return &groups[gi].Sessions[si]
			}
		}
	}
	return nil
}

func findProjectForSession(groups []ProjectGroup, name string) (*ProjectGroup, *SessionItem) {
	for gi := range groups {
		for si := range groups[gi].Sessions {
			if groups[gi].Sessions[si].Name == name {
				return &groups[gi], &groups[gi].Sessions[si]
			}
		}
	}
	return nil, nil
}
