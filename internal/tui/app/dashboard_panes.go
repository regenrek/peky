package app

import "strings"

func collectDashboardColumns(projects []ProjectGroup) []DashboardProjectColumn {
	columns := make([]DashboardProjectColumn, 0, len(projects))
	for _, project := range projects {
		column := DashboardProjectColumn{
			ProjectID:   project.ID,
			ProjectName: project.Name,
			ProjectPath: project.Path,
		}
		for _, session := range project.Sessions {
			if session.Status == StatusStopped {
				continue
			}
			for _, pane := range session.Panes {
				column.Panes = append(column.Panes, DashboardPane{
					ProjectID:   project.ID,
					ProjectName: project.Name,
					ProjectPath: project.Path,
					SessionName: session.Name,
					Pane:        pane,
				})
			}
		}
		columns = append(columns, column)
	}
	return columns
}

func dashboardSelectedProject(columns []DashboardProjectColumn, selection selectionState) string {
	if len(columns) == 0 {
		return ""
	}
	if selection.ProjectID != "" {
		for _, column := range columns {
			if column.ProjectID == selection.ProjectID {
				return column.ProjectID
			}
		}
	}
	if selection.Session != "" {
		for _, column := range columns {
			for _, pane := range column.Panes {
				if pane.SessionName == selection.Session {
					return column.ProjectID
				}
			}
		}
	}
	return columns[0].ProjectID
}

func dashboardPaneIndex(panes []DashboardPane, desired selectionState) int {
	if len(panes) == 0 {
		return -1
	}
	if strings.TrimSpace(desired.Session) != "" {
		if desired.Pane != "" {
			for i, pane := range panes {
				if pane.SessionName == desired.Session && pane.Pane.Index == desired.Pane {
					return i
				}
			}
		}
		for i, pane := range panes {
			if pane.SessionName == desired.Session {
				return i
			}
		}
	}
	return -1
}
