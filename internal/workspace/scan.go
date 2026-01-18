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

// ScanProjects scans roots for git projects and optionally non-git folders.
func ScanProjects(roots []string, allowNonGit bool) []Project {
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
		info, err := os.Stat(root)
		if err != nil || info == nil || !info.IsDir() {
			continue
		}
		scanGitProjects(root, seen, &projects)
		if allowNonGit {
			scanNonGitProjects(root, seen, &projects)
		}
	}
	return projects
}

// ScanGitProjects scans roots for git projects.
func ScanGitProjects(roots []string) []Project {
	return ScanProjects(roots, false)
}

func scanGitProjects(root string, seen map[string]struct{}, projects *[]Project) {
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
			*projects = append(*projects, Project{
				ID:     id,
				Name:   name,
				Path:   path,
				IsGit:  true,
				Source: "scan",
			})
			return filepath.SkipDir
		}
		return nil
	})
}

func scanNonGitProjects(root string, seen map[string]struct{}, projects *[]Project) {
	if hasProjectConfig(root) && !IsGitProjectPath(root) && !shouldSkipDirName(filepath.Base(root)) {
		addProjectIfMissing(root, filepath.Base(root), false, seen, projects)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry == nil || !entry.IsDir() {
			continue
		}
		if shouldSkipDir(entry) {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if IsGitProjectPath(path) {
			continue
		}
		addProjectIfMissing(path, entry.Name(), false, seen, projects)
	}
}

func addProjectIfMissing(path, name string, isGit bool, seen map[string]struct{}, projects *[]Project) {
	id := ProjectID(path, name)
	if id == "" {
		return
	}
	if _, ok := seen[id]; ok {
		return
	}
	seen[id] = struct{}{}
	*projects = append(*projects, Project{
		ID:     id,
		Name:   name,
		Path:   path,
		IsGit:  isGit,
		Source: "scan",
	})
}

func shouldSkipDir(d os.DirEntry) bool {
	if d == nil || !d.IsDir() {
		return false
	}
	return shouldSkipDirName(d.Name())
}

func isGitProjectDir(path string, d os.DirEntry) bool {
	if d == nil || !d.IsDir() || d.Name() == ".git" {
		return false
	}
	return IsGitProjectPath(path)
}

// IsGitProjectPath returns true if the path contains a .git directory.
func IsGitProjectPath(path string) bool {
	gitPath := filepath.Join(path, ".git")
	_, err := os.Stat(gitPath)
	return err == nil
}

func hasProjectConfig(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".peky.yml"))
	if err == nil {
		return true
	}
	if !os.IsNotExist(err) {
		return false
	}
	_, err = os.Stat(filepath.Join(path, ".peky.yaml"))
	return err == nil
}

func shouldSkipDirName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	if _, ok := defaultSkipDirs[name]; ok {
		return true
	}
	return false
}
