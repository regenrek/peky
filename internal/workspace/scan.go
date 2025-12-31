package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

var defaultSkipDirs = map[string]struct{}{
	"node_modules": {},
	"vendor":       {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
}

// ScanGitProjects scans roots for git projects.
func ScanGitProjects(roots []string) []Project {
	roots = NormalizeProjectRoots(roots)
	if len(roots) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	projects := make([]Project, 0, 64)
	for _, root := range roots {
		root := root
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if shouldSkipDir(d) {
				return filepath.SkipDir
			}
			if isGitProjectDir(path, d) {
				rel, _ := filepath.Rel(root, path)
				name := strings.TrimSpace(rel)
				if name == "" || name == "." {
					name = filepath.Base(path)
				}
				id := ProjectID(path, name)
				if id == "" {
					return filepath.SkipDir
				}
				if _, ok := seen[id]; ok {
					return filepath.SkipDir
				}
				seen[id] = struct{}{}
				projects = append(projects, Project{
					ID:     id,
					Name:   name,
					Path:   path,
					Source: "scan",
				})
				return filepath.SkipDir
			}
			return nil
		})
	}
	return projects
}

func shouldSkipDir(d os.DirEntry) bool {
	if d == nil || !d.IsDir() {
		return false
	}
	name := d.Name()
	if strings.HasPrefix(name, ".") {
		return true
	}
	if _, ok := defaultSkipDirs[name]; ok {
		return true
	}
	return false
}

func isGitProjectDir(path string, d os.DirEntry) bool {
	if d == nil || !d.IsDir() || d.Name() == ".git" {
		return false
	}
	gitPath := filepath.Join(path, ".git")
	_, err := os.Stat(gitPath)
	return err == nil
}
