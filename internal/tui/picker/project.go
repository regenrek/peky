package picker

// ProjectItem represents a project directory with .git.
type ProjectItem struct {
	Name        string
	Path        string
	DisplayPath string
	IsGit       bool
}

func (p ProjectItem) Title() string {
	if p.IsGit {
		return "📁 " + p.Name
	}
	return "📁 " + p.Name + " (no git)"
}
func (p ProjectItem) Description() string {
	if p.DisplayPath != "" {
		return p.DisplayPath
	}
	return p.Path
}
func (p ProjectItem) FilterValue() string { return p.Name }
