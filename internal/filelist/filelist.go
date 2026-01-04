package filelist

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/regenrek/peakypanes/internal/userpath"
)

// Config controls directory listing behavior.
type Config struct {
	MaxDepth      int
	MaxItems      int
	IncludeHidden bool
}

// Entry is a single file or directory listing.
type Entry struct {
	Path  string
	IsDir bool
}

// List returns a filtered directory listing rooted at path.
func List(path string, cfg Config) ([]Entry, bool, error) {
	root, err := sanitizeRoot(path)
	if err != nil {
		return nil, false, err
	}
	matcher := newIgnoreMatcher(root)
	entries := make([]Entry, 0, 64)
	truncated := false
	err = filepath.WalkDir(root, func(curr string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if curr == root {
			return nil
		}
		rel, relErr := filepath.Rel(root, curr)
		if relErr != nil {
			return nil
		}
		if shouldSkip(rel, d.IsDir(), cfg.IncludeHidden) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if matcher.shouldIgnore(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if cfg.MaxDepth > 0 && depth(rel) > cfg.MaxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		entries = append(entries, Entry{
			Path:  filepath.ToSlash(rel),
			IsDir: d.IsDir(),
		})
		if cfg.MaxItems > 0 && len(entries) >= cfg.MaxItems {
			truncated = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, false, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Path < entries[j].Path
	})
	return entries, truncated, nil
}

func sanitizeRoot(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("empty root path")
	}
	path = userpath.ExpandUser(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("root path is not a directory")
	}
	return abs, nil
}

func shouldSkip(rel string, isDir bool, includeHidden bool) bool {
	if includeHidden {
		return false
	}
	if rel == "" || rel == "." {
		return false
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func depth(rel string) int {
	if rel == "" || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}

type ignoreMatcher struct {
	root    string
	ignores map[string]ignore.IgnoreParser
}

func newIgnoreMatcher(root string) *ignoreMatcher {
	return &ignoreMatcher{
		root:    root,
		ignores: make(map[string]ignore.IgnoreParser),
	}
}

func (m *ignoreMatcher) shouldIgnore(rel string, isDir bool) bool {
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." {
		return false
	}
	dir := filepath.Dir(filepath.Join(m.root, filepath.FromSlash(rel)))
	relDir, err := filepath.Rel(m.root, dir)
	if err != nil {
		relDir = "."
	}
	parser := m.getIgnore(dir)
	if parser.MatchesPath(rel) {
		return true
	}
	if isDir && parser.MatchesPath(rel+"/") {
		return true
	}
	if m.matchesParents(rel, relDir) {
		return true
	}
	return false
}

func (m *ignoreMatcher) matchesParents(rel string, relDir string) bool {
	parent := relDir
	for parent != "." && parent != string(filepath.Separator) {
		parentPath := filepath.Join(m.root, parent)
		if m.getIgnore(parentPath).MatchesPath(rel) {
			return true
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parent = next
	}
	return false
}

func (m *ignoreMatcher) getIgnore(dir string) ignore.IgnoreParser {
	if parser, ok := m.ignores[dir]; ok {
		return parser
	}
	name := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(name)
	if err != nil {
		parser := ignore.CompileIgnoreLines()
		m.ignores[dir] = parser
		return parser
	}
	lines := strings.Split(string(data), "\n")
	parser := ignore.CompileIgnoreLines(lines...)
	m.ignores[dir] = parser
	return parser
}
