package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) syncExpandedSessions() {
	if m.expandedSessions == nil {
		m.expandedSessions = make(map[string]bool)
	}
	current := make(map[string]struct{})
	for _, project := range m.data.Projects {
		for _, session := range project.Sessions {
			current[session.Name] = struct{}{}
			if _, ok := m.expandedSessions[session.Name]; !ok {
				m.expandedSessions[session.Name] = true
			}
		}
	}
	for name := range m.expandedSessions {
		if _, ok := current[name]; !ok {
			delete(m.expandedSessions, name)
		}
	}
}

func (m *Model) rememberSelection(sel selectionState) {
	if sel.ProjectID == "" {
		return
	}
	if m.selectionByProject == nil {
		m.selectionByProject = make(map[string]selectionState)
	}
	m.selectionByProject[sel.ProjectID] = sel
}

func (m *Model) selectionForProjectID(projectID string) selectionState {
	if projectID == "" {
		return selectionState{}
	}
	if m.selectionByProject != nil {
		if sel, ok := m.selectionByProject[projectID]; ok {
			sel.ProjectID = projectID
			return sel
		}
	}
	return selectionState{ProjectID: projectID}
}

func (m *Model) applySelection(sel selectionState) {
	m.selection = sel
	m.rememberSelection(sel)
	if m.terminalFocus && !m.supportsTerminalFocus() {
		m.setTerminalFocus(false)
	}
}

func (m *Model) refreshSelectionForProjectConfig() bool {
	project := m.selectedProject()
	if project == nil {
		return false
	}
	projectPath := normalizeProjectPath(project.Path)
	if projectPath == "" {
		return false
	}
	if m.projectConfigState == nil {
		m.projectConfigState = make(map[string]projectConfigState)
	}
	state := projectConfigStateForPath(projectPath)
	prev, ok := m.projectConfigState[projectPath]
	m.projectConfigState[projectPath] = state
	m.updateProjectLocalConfig(projectPath, state)
	if !ok || prev.equal(state) {
		return false
	}

	delete(m.selectionByProject, project.ID)
	desired := selectionState{ProjectID: project.ID}
	var resolved selectionState
	if m.tab == TabDashboard {
		resolved = resolveDashboardSelection(m.data.Projects, desired)
		if resolved.ProjectID == "" {
			resolved = resolveSelection(m.data.Projects, desired)
		}
	} else {
		resolved = resolveSelection(m.data.Projects, desired)
	}
	if resolved.ProjectID == "" {
		return false
	}
	m.applySelection(resolved)
	m.selectionVersion++
	return true
}

func (m *Model) selectTab(delta int) {
	total := len(m.data.Projects) + 1
	if total <= 1 {
		m.tab = TabDashboard
		return
	}

	if m.tab == TabDashboard {
		m.tab = TabProject
		projectID := m.data.Projects[0].ID
		if delta < 0 {
			projectID = m.data.Projects[len(m.data.Projects)-1].ID
		}
		resolved := resolveSelection(m.data.Projects, m.selectionForProjectID(projectID))
		m.applySelection(resolved)
		m.selectionVersion++
		return
	}

	idx, ok := m.projectIndexForID(m.selection.ProjectID)
	if !ok {
		idx = 0
	}
	current := idx + 1
	next := wrapIndex(current+delta, total)
	if next == 0 {
		m.tab = TabDashboard
		m.selectionVersion++
		return
	}
	projectID := m.data.Projects[next-1].ID
	resolved := resolveSelection(m.data.Projects, m.selectionForProjectID(projectID))
	m.applySelection(resolved)
	m.selectionVersion++
}

func (m *Model) selectDashboardTab() bool {
	if m.tab == TabDashboard {
		return false
	}
	m.tab = TabDashboard
	m.selectionVersion++
	return true
}

func (m *Model) selectProjectTab(projectID string) bool {
	if strings.TrimSpace(projectID) == "" {
		return false
	}
	project := findProjectByID(m.data.Projects, projectID)
	if project == nil {
		return false
	}
	resolved := resolveSelection(m.data.Projects, m.selectionForProjectID(project.ID))
	changed := m.tab != TabProject || m.selection != resolved
	m.tab = TabProject
	m.applySelection(resolved)
	if changed {
		m.selectionVersion++
	}
	return changed
}

func (m *Model) selectSession(delta int) {
	project := m.selectedProject()
	if project == nil || len(project.Sessions) == 0 {
		return
	}
	filtered := m.filteredSessions(project.Sessions)
	if len(filtered) == 0 {
		return
	}
	idx := sessionIndex(filtered, m.selection.Session)
	idx = wrapIndex(idx+delta, len(filtered))
	m.selection.Session = filtered[idx].Name
	m.selection.Pane = ""
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

func (m *Model) selectSessionOrPane(delta int) {
	project := m.selectedProject()
	if project == nil || len(project.Sessions) == 0 {
		return
	}
	filtered := m.filteredSessions(project.Sessions)
	if len(filtered) == 0 {
		return
	}
	items := buildSessionPaneEntries(filtered, m.sessionExpanded)
	if len(items) == 0 {
		return
	}

	current := findSessionPaneIndex(items, m.selection)
	next := items[wrapIndex(current+delta, len(items))]
	m.selection.Session = next.session
	m.selection.Pane = next.pane
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

type sessionPaneEntry struct {
	session string
	pane    string
}

func buildSessionPaneEntries(sessions []SessionItem, expanded func(string) bool) []sessionPaneEntry {
	items := make([]sessionPaneEntry, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, sessionPaneEntry{session: session.Name})
		if !expanded(session.Name) {
			continue
		}
		for _, pane := range session.Panes {
			items = append(items, sessionPaneEntry{session: session.Name, pane: pane.Index})
		}
	}
	return items
}

func findSessionPaneIndex(items []sessionPaneEntry, selection selectionState) int {
	if selection.Pane != "" {
		if idx := matchSessionPane(items, selection.Session, selection.Pane); idx >= 0 {
			return idx
		}
	}
	if selection.Session != "" {
		if idx := matchSessionPane(items, selection.Session, ""); idx >= 0 {
			return idx
		}
	}
	return 0
}

func matchSessionPane(items []sessionPaneEntry, session, pane string) int {
	for i, item := range items {
		if item.session == session && item.pane == pane {
			return i
		}
	}
	return -1
}

func (m *Model) selectDashboardPane(delta int) {
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return
	}
	filtered := m.filteredDashboardColumns(columns)
	if len(filtered) == 0 {
		return
	}
	projectIndex := m.dashboardProjectIndex(filtered)
	if projectIndex < 0 || projectIndex >= len(filtered) {
		projectIndex = 0
	}
	column := filtered[projectIndex]
	if len(column.Panes) == 0 {
		return
	}
	idx := dashboardPaneIndex(column.Panes, m.selection)
	if idx < 0 {
		idx = 0
	}
	idx = wrapIndex(idx+delta, len(column.Panes))
	pane := column.Panes[idx]
	m.selection.ProjectID = column.ProjectID
	m.selection.Session = pane.SessionName
	m.selection.Pane = pane.Pane.Index
	m.selectionVersion++
	m.rememberSelection(m.selection)
}

func (m *Model) selectDashboardProject(delta int) {
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return
	}
	filtered := m.filteredDashboardColumns(columns)
	if len(filtered) == 0 {
		return
	}
	idx := m.dashboardProjectIndex(filtered)
	if idx < 0 {
		idx = 0
	}
	idx = wrapIndex(idx+delta, len(filtered))
	target := filtered[idx]
	desired := m.selectionForProjectID(target.ProjectID)
	resolved := resolveDashboardSelectionFromColumns(filtered, desired)
	if resolved.ProjectID == "" {
		resolved.ProjectID = target.ProjectID
	}
	m.applySelection(resolved)
	m.selectionVersion++
}

func (m *Model) selectPane(delta int) {
	session := m.selectedSession()
	if session == nil {
		return
	}
	if len(session.Panes) == 0 {
		return
	}

	currentPane := strings.TrimSpace(m.selection.Pane)
	if currentPane == "" {
		if active := activePaneIndex(session.Panes); active != "" {
			currentPane = active
		} else {
			currentPane = session.Panes[0].Index
		}
	}

	idx := -1
	for i, pane := range session.Panes {
		if pane.Index == currentPane {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 0
	}
	idx = wrapIndex(idx+delta, len(session.Panes))
	next := session.Panes[idx]
	m.selection.Pane = next.Index
	m.rememberSelection(m.selection)
}

func (m *Model) togglePanes() {
	session := m.selectedSession()
	if session == nil {
		return
	}
	current := m.expandedSessions[session.Name]
	m.expandedSessions[session.Name] = !current
}

func (m *Model) cyclePane(delta int) tea.Cmd {
	prevPane := m.selection.Pane
	m.selectPane(delta)
	changed := m.selection.Pane != prevPane
	if !changed {
		return nil
	}
	m.selectionVersion++
	return nil
}

func (m *Model) selectedProject() *ProjectGroup {
	if m.tab == TabDashboard {
		if project, _ := findProjectForSession(m.data.Projects, m.selection.Session); project != nil {
			return project
		}
	}
	for i := range m.data.Projects {
		if m.data.Projects[i].ID == m.selection.ProjectID {
			return &m.data.Projects[i]
		}
	}
	if len(m.data.Projects) > 0 {
		return &m.data.Projects[0]
	}
	return nil
}

func (m *Model) selectedSession() *SessionItem {
	if m.tab == TabDashboard {
		if _, session := findProjectForSession(m.data.Projects, m.selection.Session); session != nil {
			return session
		}
		for i := range m.data.Projects {
			if len(m.data.Projects[i].Sessions) > 0 {
				return &m.data.Projects[i].Sessions[0]
			}
		}
		return nil
	}
	project := m.selectedProject()
	if project == nil {
		return nil
	}
	for i := range project.Sessions {
		if project.Sessions[i].Name == m.selection.Session {
			return &project.Sessions[i]
		}
	}
	if len(project.Sessions) > 0 {
		return &project.Sessions[0]
	}
	return nil
}

func (m *Model) selectedPane() *PaneItem {
	session := m.selectedSession()
	if session == nil {
		return nil
	}
	if len(session.Panes) == 0 {
		return nil
	}
	if m.selection.Pane != "" {
		for i := range session.Panes {
			if session.Panes[i].Index == m.selection.Pane {
				return &session.Panes[i]
			}
		}
	}
	for i := range session.Panes {
		if session.Panes[i].Active {
			return &session.Panes[i]
		}
	}
	return &session.Panes[0]
}
