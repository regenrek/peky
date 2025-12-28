package peakypanes

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
	Kind        headerPartKind
	Label       string
	ProjectName string
	Rendered    string
	Width       int
}

type headerHit struct {
	Kind        headerPartKind
	ProjectName string
}

type headerHitRect struct {
	Hit  headerHit
	Rect rect
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
		activeProject := m.headerActiveProjectName()
		for _, p := range m.data.Projects {
			style := theme.TabInactive
			if m.tab == TabProject && p.Name == activeProject {
				style = theme.TabActive
			}
			rendered := style.Render(p.Name)
			parts = append(parts, headerPart{
				Kind:        headerPartProject,
				Label:       p.Name,
				ProjectName: p.Name,
				Rendered:    rendered,
				Width:       lipgloss.Width(rendered),
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

func (m Model) headerActiveProjectName() string {
	activeProject := m.selection.Project
	if m.tab != TabProject {
		return activeProject
	}
	found := false
	for _, p := range m.data.Projects {
		if p.Name == activeProject {
			found = true
			break
		}
	}
	if !found && len(m.data.Projects) > 0 {
		activeProject = m.data.Projects[0].Name
	}
	return activeProject
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
