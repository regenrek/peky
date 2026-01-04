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

type listWalker struct {
	root      string
	cfg       Config
	matcher   *ignoreMatcher
	entries   []Entry
	truncated bool
}

// List returns a filtered directory listing rooted at path.
func List(path string, cfg Config) ([]Entry, bool, error) {
	root, err := sanitizeRoot(path)
	if err != nil {
		return nil, false, err
	}
	walker := &listWalker{
		root:    root,
		cfg:     cfg,
		matcher: newIgnoreMatcher(root),
		entries: make([]Entry, 0, 64),
	}
	err = filepath.WalkDir(root, walker.visit)
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, false, err
	}
	sort.SliceStable(walker.entries, func(i, j int) bool {
		if walker.entries[i].IsDir != walker.entries[j].IsDir {
			return walker.entries[i].IsDir
		}
		return walker.entries[i].Path < walker.entries[j].Path
	})
	return walker.entries, walker.truncated, nil
}

func (w *listWalker) visit(curr string, d os.DirEntry, walkErr error) error {
	if walkErr != nil {
		return nil
	}
	rel, ok := w.relPath(curr)
	if !ok {
		return nil
	}
	if w.skipHidden(rel, d) {
		return w.skipDir(d)
	}
	if w.skipIgnored(rel, d) {
		return w.skipDir(d)
	}
	if w.skipDepth(rel, d) {
		return w.skipDir(d)
	}
	w.entries = append(w.entries, Entry{
		Path:  filepath.ToSlash(rel),
		IsDir: d.IsDir(),
	})
	if w.cfg.MaxItems > 0 && len(w.entries) >= w.cfg.MaxItems {
		w.truncated = true
		return filepath.SkipAll
	}
	return nil
}

func (w *listWalker) relPath(curr string) (string, bool) {
	if curr == w.root {
		return "", false
	}
	rel, err := filepath.Rel(w.root, curr)
	if err != nil {
		return "", false
	}
	return rel, true
}

func (w *listWalker) skipHidden(rel string, d os.DirEntry) bool {
	return shouldSkip(rel, d.IsDir(), w.cfg.IncludeHidden)
}

func (w *listWalker) skipIgnored(rel string, d os.DirEntry) bool {
	return w.matcher.shouldIgnore(rel, d.IsDir())
}

func (w *listWalker) skipDepth(rel string, d os.DirEntry) bool {
	if w.cfg.MaxDepth <= 0 {
		return false
	}
	return depth(rel) > w.cfg.MaxDepth
}

func (w *listWalker) skipDir(d os.DirEntry) error {
	if d.IsDir() {
		return filepath.SkipDir
	}
	return nil
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
