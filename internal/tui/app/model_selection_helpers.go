package app

import "strings"

func (m *Model) filteredSessions(sessions []SessionItem) []SessionItem {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return sessions
	}
	filter = strings.ToLower(filter)
	var out []SessionItem
	for _, s := range sessions {
		if strings.Contains(strings.ToLower(s.Name), filter) || strings.Contains(strings.ToLower(s.Path), filter) {
			out = append(out, s)
		}
	}
	return out
}

func (m *Model) filteredDashboardColumns(columns []DashboardProjectColumn) []DashboardProjectColumn {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return columns
	}
	filter = strings.ToLower(filter)
	out := make([]DashboardProjectColumn, 0, len(columns))
	for _, column := range columns {
		next := DashboardProjectColumn{
			ProjectName: column.ProjectName,
			ProjectPath: column.ProjectPath,
		}
		for _, pane := range column.Panes {
			if strings.Contains(strings.ToLower(pane.ProjectName), filter) ||
				strings.Contains(strings.ToLower(pane.ProjectPath), filter) ||
				strings.Contains(strings.ToLower(pane.SessionName), filter) ||
				strings.Contains(strings.ToLower(pane.Pane.Title), filter) ||
				strings.Contains(strings.ToLower(pane.Pane.Command), filter) ||
				strings.Contains(strings.ToLower(pane.Pane.Index), filter) {
				next.Panes = append(next.Panes, pane)
			}
		}
		out = append(out, next)
	}
	return out
}

func (m *Model) projectIndexFor(name string) (int, bool) {
	for i := range m.data.Projects {
		if m.data.Projects[i].Name == name {
			return i, true
		}
	}
	return -1, false
}

func (m *Model) dashboardProjectIndex(columns []DashboardProjectColumn) int {
	if len(columns) == 0 {
		return -1
	}
	if m.selection.Project != "" {
		for i, column := range columns {
			if column.ProjectName == m.selection.Project {
				return i
			}
		}
	}
	if m.selection.Session != "" {
		for i, column := range columns {
			for _, pane := range column.Panes {
				if pane.SessionName == m.selection.Session {
					return i
				}
			}
		}
	}
	return 0
}

func sessionIndex(sessions []SessionItem, name string) int {
	for i := range sessions {
		if sessions[i].Name == name {
			return i
		}
	}
	return 0
}

func wrapIndex(idx, total int) int {
	if total <= 0 {
		return 0
	}
	if idx < 0 {
		return total - 1
	}
	if idx >= total {
		return 0
	}
	return idx
}
