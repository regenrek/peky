package app

import (
	"fmt"
	"strings"
)

func (m *Model) pekyContext() string {
	if m == nil {
		return ""
	}
	var lines []string
	project := m.selectedProject()
	session := m.selectedSession()
	pane := m.selectedPane()
	if project != nil {
		lines = append(lines, fmt.Sprintf("Selected project: %s (%s)", project.Name, project.Path))
	}
	if session != nil {
		lines = append(lines, fmt.Sprintf("Selected session: %s", session.Name))
	}
	if pane != nil {
		cwd := strings.TrimSpace(pane.Cwd)
		if cwd == "" {
			cwd = "unknown"
		}
		title := strings.TrimSpace(pane.Title)
		if title == "" {
			title = "pane " + pane.Index
		}
		lines = append(lines, fmt.Sprintf("Selected pane: id=%s index=%s title=%s cwd=%s", pane.ID, pane.Index, title, cwd))
		lines = append(lines, fmt.Sprintf("Use --pane-id %s for pane-specific commands.", pane.ID))
	}
	if len(lines) == 0 {
		return ""
	}
	return "Context:\n" + strings.Join(lines, "\n")
}
