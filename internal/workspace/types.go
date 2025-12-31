package workspace

// Project describes a workspace project.
type Project struct {
	ID     string
	Name   string
	Path   string
	Hidden bool
	Source string
}

// Workspace describes the workspace roots and projects.
type Workspace struct {
	Roots    []string
	Projects []Project
}

// ProjectRef identifies a project by id, name, or path.
type ProjectRef struct {
	ID   string
	Name string
	Path string
}

// Ref returns a reference for the project.
func (p Project) Ref() ProjectRef {
	return ProjectRef{ID: p.ID, Name: p.Name, Path: p.Path}
}
