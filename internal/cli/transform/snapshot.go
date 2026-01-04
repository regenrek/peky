package transform

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/workspace"
)

// SessionSummaries builds session summaries for list output.
func SessionSummaries(sessions []native.SessionSnapshot) []output.SessionSummary {
	out := make([]output.SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		lastActivity := session.CreatedAt
		for _, pane := range session.Panes {
			if pane.LastActive.After(lastActivity) {
				lastActivity = pane.LastActive
			}
		}
		out = append(out, output.SessionSummary{
			Name:         session.Name,
			Layout:       session.LayoutName,
			Cwd:          session.Path,
			PaneCount:    len(session.Panes),
			CreatedAt:    session.CreatedAt,
			LastActivity: lastActivity,
		})
	}
	return out
}

// PaneList builds pane summaries with project context.
func PaneList(sessions []native.SessionSnapshot, ws workspace.Workspace, filterSession string) []output.PaneSummaryWithContext {
	filterSession = strings.TrimSpace(filterSession)
	projectByPath := projectIndexByPath(ws)
	out := make([]output.PaneSummaryWithContext, 0, len(sessions)*2)
	for _, session := range sessions {
		if filterSession != "" && session.Name != filterSession {
			continue
		}
		project := projectByPath[workspace.NormalizeProjectPath(session.Path)]
		for _, pane := range session.Panes {
			out = append(out, output.PaneSummaryWithContext{
				PaneSummary: paneSummaryFromNative(pane),
				SessionName: session.Name,
				ProjectID:   project.ID,
				ProjectName: project.Name,
			})
		}
	}
	return out
}

// BuildSnapshot groups sessions into projects for snapshot output.
func BuildSnapshot(sessions []native.SessionSnapshot, ws workspace.Workspace, focusedSession, focusedPane string) output.Snapshot {
	projectByPath := projectIndexByPath(ws)
	groups := make(map[string]*output.ProjectSnapshot)
	for _, project := range ws.Projects {
		groups[project.ID] = &output.ProjectSnapshot{
			ID:       project.ID,
			Name:     project.Name,
			Path:     project.Path,
			Hidden:   project.Hidden,
			Sessions: nil,
		}
	}
	for _, session := range sessions {
		pathKey := workspace.NormalizeProjectPath(session.Path)
		project := projectByPath[pathKey]
		if project.ID == "" {
			project = workspace.Project{
				ID:     workspace.ProjectID(session.Path, session.Name),
				Name:   groupNameFromPath(session.Path, session.Name),
				Path:   session.Path,
				Hidden: false,
			}
		}
		group := groups[project.ID]
		if group == nil {
			group = &output.ProjectSnapshot{
				ID:     project.ID,
				Name:   project.Name,
				Path:   project.Path,
				Hidden: project.Hidden,
			}
			groups[project.ID] = group
		}
		group.Sessions = append(group.Sessions, sessionSnapshotFromNative(session))
	}
	projects := make([]output.ProjectSnapshot, 0, len(groups))
	for _, project := range groups {
		projects = append(projects, *project)
	}
	sort.Slice(projects, func(i, j int) bool {
		if strings.EqualFold(projects[i].Name, projects[j].Name) {
			return projects[i].ID < projects[j].ID
		}
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
	selectedProject, selectedSession := resolveSelection(projects, focusedSession)
	return output.Snapshot{
		Projects:        projects,
		SelectedProject: selectedProject,
		SelectedSession: selectedSession,
		SelectedPaneID:  strings.TrimSpace(focusedPane),
		GeneratedAt:     time.Now().UTC(),
	}
}

func sessionSnapshotFromNative(session native.SessionSnapshot) output.SessionSnapshot {
	panes := make([]output.PaneSummary, 0, len(session.Panes))
	for _, pane := range session.Panes {
		panes = append(panes, paneSummaryFromNative(pane))
	}
	return output.SessionSnapshot{
		Name:   session.Name,
		Layout: session.LayoutName,
		Cwd:    session.Path,
		Panes:  panes,
	}
}

func paneSummaryFromNative(pane native.PaneSnapshot) output.PaneSummary {
	index := parseIndex(pane.Index)
	return output.PaneSummary{
		ID:           pane.ID,
		Index:        index,
		Title:        pane.Title,
		Command:      pane.Command,
		StartCmd:     pane.StartCommand,
		Tool:         pane.Tool,
		Cwd:          pane.Cwd,
		Dead:         pane.Dead,
		Tags:         append([]string(nil), pane.Tags...),
		LastActivity: pane.LastActive,
		BytesIn:      pane.BytesIn,
		BytesOut:     pane.BytesOut,
	}
}

func parseIndex(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func projectIndexByPath(ws workspace.Workspace) map[string]workspace.Project {
	out := make(map[string]workspace.Project, len(ws.Projects))
	for _, project := range ws.Projects {
		key := workspace.NormalizeProjectPath(project.Path)
		if key != "" {
			out[key] = project
		}
	}
	return out
}

func groupNameFromPath(path, fallback string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return fallback
	}
	base := filepath.Base(path)
	base = strings.TrimSpace(base)
	if base == "" || base == string(filepath.Separator) || base == "." {
		return fallback
	}
	return base
}

func resolveSelection(projects []output.ProjectSnapshot, focusedSession string) (string, string) {
	focusedSession = strings.TrimSpace(focusedSession)
	if focusedSession != "" {
		for _, project := range projects {
			for _, session := range project.Sessions {
				if session.Name == focusedSession {
					return project.ID, focusedSession
				}
			}
		}
	}
	if len(projects) == 1 {
		project := projects[0]
		if len(project.Sessions) == 1 {
			return project.ID, project.Sessions[0].Name
		}
		return project.ID, ""
	}
	return "", ""
}
