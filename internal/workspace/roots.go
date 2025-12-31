package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/userpath"
)

// DefaultProjectRoots returns the default project roots.
func DefaultProjectRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "projects")}
}

// NormalizeProjectPath expands user paths and cleans them.
func NormalizeProjectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = userpath.ExpandUser(path)
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// NormalizeProjectRoots normalizes and deduplicates root paths.
func NormalizeProjectRoots(roots []string) []string {
	if len(roots) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		root = NormalizeProjectPath(root)
		if root == "" {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}

// ResolveProjectRoots returns normalized roots or defaults.
func ResolveProjectRoots(roots []string) []string {
	resolved := NormalizeProjectRoots(roots)
	if len(resolved) == 0 {
		return DefaultProjectRoots()
	}
	return resolved
}

// ProjectID returns the stable project id (path preferred, else name).
func ProjectID(path, name string) string {
	path = NormalizeProjectPath(path)
	if path != "" {
		return strings.ToLower(path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return strings.ToLower(name)
}
