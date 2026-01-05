package app

import "time"

func (m *Model) mergePanePreviews(prev DashboardData) {
	if m == nil || len(prev.Projects) == 0 || len(m.data.Projects) == 0 {
		return
	}
	prevPreview := collectPanePreviews(prev)
	if len(prevPreview) == 0 {
		return
	}
	applyPanePreviews(m.data.Projects, prevPreview, m.settings, time.Now())
}

func collectPanePreviews(prev DashboardData) map[string][]string {
	prevPreview := make(map[string][]string)
	for _, project := range prev.Projects {
		for _, session := range project.Sessions {
			for _, pane := range session.Panes {
				if pane.ID == "" || len(pane.Preview) == 0 {
					continue
				}
				prevPreview[pane.ID] = pane.Preview
			}
		}
	}
	return prevPreview
}

func applyPanePreviews(projects []ProjectGroup, previews map[string][]string, settings DashboardConfig, now time.Time) {
	for projIdx := range projects {
		project := &projects[projIdx]
		for sessIdx := range project.Sessions {
			session := &project.Sessions[sessIdx]
			for paneIdx := range session.Panes {
				pane := &session.Panes[paneIdx]
				if pane.ID == "" || pane.Disconnected || pane.Dead {
					continue
				}
				if len(pane.Preview) > 0 {
					continue
				}
				preview, ok := previews[pane.ID]
				if !ok || len(preview) == 0 {
					continue
				}
				pane.Preview = append([]string(nil), preview...)
				pane.Status = classifyPane(*pane, pane.Preview, settings, now)
			}
		}
	}
}
