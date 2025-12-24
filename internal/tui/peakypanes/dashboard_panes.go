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
			for _, window := range session.Windows {
				if len(window.Panes) == 0 {
					continue
				}
				for _, pane := range window.Panes {
					column.Panes = append(column.Panes, DashboardPane{
						ProjectName: project.Name,
						ProjectPath: project.Path,
						SessionName: session.Name,
						WindowIndex: window.Index,
						WindowName:  window.Name,
						Pane:        pane,
					})
				}
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
		if desired.Window != "" && desired.Pane != "" {
			for i, pane := range panes {
				if pane.SessionName == desired.Session && pane.WindowIndex == desired.Window && pane.Pane.Index == desired.Pane {
					return i
				}
			}
		}
		if desired.Window != "" {
			for i, pane := range panes {
				if pane.SessionName == desired.Session && pane.WindowIndex == desired.Window {
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
