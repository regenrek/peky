package workspace

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
)

// ListWorkspace builds the workspace project list from config + scanned roots.
func ListWorkspace(configPath string) (Workspace, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return Workspace{}, err
	}
	roots := ResolveProjectRoots(cfg.Dashboard.ProjectRoots)
	hidden := HiddenProjectKeySet(cfg.Dashboard.HiddenProjects)
	projects := make(map[string]Project)
	for _, projectCfg := range cfg.Projects {
		project := projectFromConfig(projectCfg)
		if project.ID == "" {
			continue
		}
		projects[project.ID] = project
	}
	for _, scanned := range ScanGitProjects(roots) {
		if scanned.ID == "" {
			continue
		}
		if _, ok := projects[scanned.ID]; ok {
			continue
		}
		projects[scanned.ID] = scanned
	}
	out := make([]Project, 0, len(projects))
	for _, project := range projects {
		project.Hidden = isHiddenProject(hidden, project)
		out = append(out, project)
	}
	sort.Slice(out, func(i, j int) bool {
		if strings.EqualFold(out[i].Name, out[j].Name) {
			return out[i].ID < out[j].ID
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return Workspace{Roots: roots, Projects: out}, nil
}

// FindProject locates a project by ref.
func FindProject(projects []Project, ref ProjectRef) (*Project, error) {
	id := strings.TrimSpace(ref.ID)
	name := strings.TrimSpace(ref.Name)
	path := NormalizeProjectPath(ref.Path)
	if id == "" && name == "" && path == "" {
		return nil, fmt.Errorf("workspace: project ref required")
	}
	for i := range projects {
		project := &projects[i]
		if id != "" && strings.EqualFold(project.ID, id) {
			return project, nil
		}
		if path != "" && NormalizeProjectPath(project.Path) == path {
			return project, nil
		}
		if name != "" && strings.EqualFold(project.Name, name) {
			return project, nil
		}
	}
	return nil, fmt.Errorf("workspace: project not found")
}

func projectFromConfig(cfg layout.ProjectConfig) Project {
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = strings.TrimSpace(cfg.Session)
	}
	path := NormalizeProjectPath(cfg.Path)
	if name == "" {
		if path != "" {
			name = filepath.Base(path)
		} else {
			name = "project"
		}
	}
	id := ProjectID(path, name)
	return Project{
		ID:     id,
		Name:   name,
		Path:   path,
		Source: "config",
	}
}

func isHiddenProject(hidden map[string]struct{}, project Project) bool {
	if len(hidden) == 0 {
		return false
	}
	path := NormalizeProjectPath(project.Path)
	if path != "" {
		if _, ok := hidden[strings.ToLower(path)]; ok {
			return true
		}
	}
	name := strings.TrimSpace(project.Name)
	if name == "" {
		return false
	}
	_, ok := hidden[strings.ToLower(name)]
	return ok
}
