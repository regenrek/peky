package picker

// ProjectItem represents a project directory with .git.
type ProjectItem struct {
	Name        string
	Path        string
	DisplayPath string
}

func (p ProjectItem) Title() string { return "📁 " + p.Name }
func (p ProjectItem) Description() string {
	if p.DisplayPath != "" {
		return p.DisplayPath
	}
	return p.Path
}
func (p ProjectItem) FilterValue() string { return p.Name }
