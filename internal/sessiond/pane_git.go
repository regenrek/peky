package sessiond

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	paneGitCacheTTL         = 2 * time.Second
	paneGitRequestQueueSize = 1024
)

type paneGitCache struct {
	mu       sync.Mutex
	byCwd    map[string]paneGitCacheEntry
	inFlight map[string]struct{}

	ttl    time.Duration
	maxLen int

	probe     func(context.Context, string) (PaneGitMeta, bool)
	requests  chan string
	startOnce sync.Once
}

type paneGitCacheEntry struct {
	meta    PaneGitMeta
	expires time.Time
}

func newPaneGitCache() *paneGitCache {
	return &paneGitCache{
		byCwd:    make(map[string]paneGitCacheEntry),
		inFlight: make(map[string]struct{}),
		ttl:      paneGitCacheTTL,
		maxLen:   4096,
		probe:    probePaneGitMeta,
	}
}

func (c *paneGitCache) Start(ctx context.Context, wg *sync.WaitGroup, workers int) {
	if c == nil {
		return
	}
	if ctx == nil {
		return
	}
	if workers <= 0 {
		workers = 1
	}
	c.startOnce.Do(func() {
		c.mu.Lock()
		if c.probe == nil {
			c.probe = probePaneGitMeta
		}
		if c.byCwd == nil {
			c.byCwd = make(map[string]paneGitCacheEntry)
		}
		if c.inFlight == nil {
			c.inFlight = make(map[string]struct{})
		}
		c.requests = make(chan string, paneGitRequestQueueSize)
		c.mu.Unlock()

		for i := 0; i < workers; i++ {
			if wg != nil {
				wg.Add(1)
			}
			go func() {
				if wg != nil {
					defer wg.Done()
				}
				c.worker(ctx)
			}()
		}
	})
}

func (c *paneGitCache) Meta(ctx context.Context, cwd string) (PaneGitMeta, bool) {
	if c == nil {
		return PaneGitMeta{}, false
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return PaneGitMeta{}, false
	}
	now := time.Now()

	c.mu.Lock()
	entry, ok := c.byCwd[cwd]
	if ok && now.Before(entry.expires) {
		c.mu.Unlock()
		return entry.meta, entry.meta.Root != ""
	}
	if c.requests != nil {
		entry.expires = now.Add(c.ttl)
		if len(c.byCwd) > c.maxLen {
			c.pruneLocked(now)
		}
		c.byCwd[cwd] = entry
		c.enqueueProbeLocked(cwd)
		meta := entry.meta
		c.mu.Unlock()
		return meta, meta.Root != ""
	}
	probe := c.probe
	c.mu.Unlock()

	meta, ok := probe(ctx, cwd)
	if ok {
		meta.UpdatedAt = now
	}

	c.mu.Lock()
	if len(c.byCwd) > c.maxLen {
		c.pruneLocked(now)
	}
	c.byCwd[cwd] = paneGitCacheEntry{meta: meta, expires: now.Add(c.ttl)}
	c.mu.Unlock()
	return meta, ok
}

func (c *paneGitCache) pruneLocked(now time.Time) {
	for key, entry := range c.byCwd {
		if now.After(entry.expires) {
			delete(c.byCwd, key)
		}
	}
	if len(c.byCwd) <= c.maxLen {
		return
	}
	for key := range c.byCwd {
		delete(c.byCwd, key)
		if len(c.byCwd) <= c.maxLen {
			return
		}
	}
}

func (c *paneGitCache) enqueueProbeLocked(cwd string) {
	if c == nil || c.requests == nil {
		return
	}
	if c.inFlight == nil {
		c.inFlight = make(map[string]struct{})
	}
	if _, ok := c.inFlight[cwd]; ok {
		return
	}
	c.inFlight[cwd] = struct{}{}
	select {
	case c.requests <- cwd:
	default:
		delete(c.inFlight, cwd)
	}
}

func (c *paneGitCache) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cwd := <-c.requests:
			c.runProbe(ctx, cwd)
		}
	}
}

func (c *paneGitCache) runProbe(ctx context.Context, cwd string) {
	if c == nil {
		return
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		c.mu.Lock()
		if c.inFlight != nil {
			delete(c.inFlight, cwd)
		}
		c.mu.Unlock()
		return
	}
	probe := c.probe
	if probe == nil {
		probe = probePaneGitMeta
	}
	meta, ok := probe(ctx, cwd)
	now := time.Now()
	if ok {
		meta.UpdatedAt = now
	}

	c.mu.Lock()
	if c.inFlight != nil {
		delete(c.inFlight, cwd)
	}
	if len(c.byCwd) > c.maxLen {
		c.pruneLocked(now)
	}
	c.byCwd[cwd] = paneGitCacheEntry{meta: meta, expires: now.Add(c.ttl)}
	c.mu.Unlock()
}

func probePaneGitMeta(parent context.Context, cwd string) (PaneGitMeta, bool) {
	ctx, cancel := context.WithTimeout(parent, 800*time.Millisecond)
	defer cancel()

	root, gitDir, commonDir, branch, ok := gitRevParse(ctx, cwd)
	if !ok || strings.TrimSpace(root) == "" {
		return PaneGitMeta{}, false
	}
	dirty := gitDirty(ctx, cwd)

	gitDirAbs := absolutePath(cwd, gitDir)
	commonDirAbs := absolutePath(cwd, commonDir)
	worktree := gitDirAbs != "" && commonDirAbs != "" && filepath.Clean(gitDirAbs) != filepath.Clean(commonDirAbs)

	return PaneGitMeta{
		Root:     root,
		Branch:   branch,
		Dirty:    dirty,
		Worktree: worktree,
	}, true
}

func absolutePath(base, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}

func gitRevParse(ctx context.Context, cwd string) (root, gitDir, commonDir, branch string, ok bool) {
	out, err := gitCmd(ctx, cwd, "rev-parse", "--show-toplevel", "--git-dir", "--git-common-dir", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", "", "", false
	}
	lines := splitNonEmptyLines(out)
	if len(lines) < 4 {
		return "", "", "", "", false
	}
	return lines[0], lines[1], lines[2], lines[3], true
}

func gitDirty(ctx context.Context, cwd string) bool {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	out, err := gitCmd(ctx, cwd, "status", "--porcelain=v1")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

func gitCmd(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func splitNonEmptyLines(s string) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
