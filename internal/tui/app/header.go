package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/regenrek/peakypanes/internal/tui/theme"
)

type headerPartKind int

const (
	headerPartLogo headerPartKind = iota
	headerPartDashboard
	headerPartProject
	headerPartPlaceholder
	headerPartNew
)

type headerPart struct {
	Kind      headerPartKind
	Label     string
	ProjectID string
	Rendered  string
	Width     int
}

func (k headerPartKind) clickable() bool {
	switch k {
	case headerPartDashboard, headerPartProject, headerPartNew:
		return true
	default:
		return false
	}
}

func (m Model) headerParts() []headerPart {
	parts := make([]headerPart, 0, len(m.data.Projects)+3)

	logo := "🎩 Peaky Panes"
	parts = append(parts, headerPart{
		Kind:     headerPartLogo,
		Label:    logo,
		Rendered: logo,
		Width:    lipgloss.Width(logo),
	})

	dashboardLabel := "Dashboard"
	dashboardStyle := theme.TabInactive
	if m.tab == TabDashboard {
		dashboardStyle = theme.TabActive
	}
	dashboardRendered := dashboardStyle.Render(dashboardLabel)
	parts = append(parts, headerPart{
		Kind:     headerPartDashboard,
		Label:    dashboardLabel,
		Rendered: dashboardRendered,
		Width:    lipgloss.Width(dashboardRendered),
	})

	if len(m.data.Projects) == 0 {
		noneRendered := theme.TabInactive.Render("none")
		parts = append(parts, headerPart{
			Kind:     headerPartPlaceholder,
			Label:    "none",
			Rendered: noneRendered,
			Width:    lipgloss.Width(noneRendered),
		})
	} else {
		activeProjectID := m.headerActiveProjectID()
		for _, p := range m.data.Projects {
			style := theme.TabInactive
			if m.tab == TabProject && p.ID == activeProjectID {
				style = theme.TabActive
			}
			rendered := style.Render(p.Name)
			parts = append(parts, headerPart{
				Kind:      headerPartProject,
				Label:     p.Name,
				ProjectID: p.ID,
				Rendered:  rendered,
				Width:     lipgloss.Width(rendered),
			})
		}
	}

	newRendered := theme.TabAdd.Render("+ New")
	parts = append(parts, headerPart{
		Kind:     headerPartNew,
		Label:    "+ New",
		Rendered: newRendered,
		Width:    lipgloss.Width(newRendered),
	})

	return parts
}

func (m Model) headerActiveProjectID() string {
	activeProjectID := m.selection.ProjectID
	if m.tab != TabProject {
		return activeProjectID
	}
	if activeProjectID != "" {
		if project := findProjectByID(m.data.Projects, activeProjectID); project != nil {
			return project.ID
		}
	}
	if len(m.data.Projects) > 0 {
		return m.data.Projects[0].ID
	}
	return ""
}

func headerLine(parts []headerPart) string {
	if len(parts) == 0 {
		return ""
	}
	var builder strings.Builder
	for i, part := range parts {
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(part.Rendered)
	}
	return builder.String()
}
