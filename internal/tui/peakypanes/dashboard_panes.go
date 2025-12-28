package peakypanes

import "strings"

func collectDashboardColumns(projects []ProjectGroup) []DashboardProjectColumn {
	columns := make([]DashboardProjectColumn, 0, len(projects))
	for _, project := range projects {
		column := DashboardProjectColumn{
			ProjectName: project.Name,
			ProjectPath: project.Path,
		}
		for _, session := range project.Sessions {
			if session.Status == StatusStopped {
				continue
			}
			for _, pane := range session.Panes {
				column.Panes = append(column.Panes, DashboardPane{
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
