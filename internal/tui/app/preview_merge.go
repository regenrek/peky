package app

import "time"

func (m *Model) mergePanePreviews(prev DashboardData) {
	if m == nil || len(prev.Projects) == 0 || len(m.data.Projects) == 0 {
		return
	}
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
	if len(prevPreview) == 0 {
		return
	}
	now := time.Now()
	for projIdx := range m.data.Projects {
		project := &m.data.Projects[projIdx]
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
				preview, ok := prevPreview[pane.ID]
				if !ok || len(preview) == 0 {
					continue
				}
				pane.Preview = append([]string(nil), preview...)
				pane.Status = classifyPane(*pane, pane.Preview, m.settings, now)
			}
		}
	}
}
