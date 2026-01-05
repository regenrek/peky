package sessionrestore

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/regenrek/peakypanes/internal/atomicfile"
	"github.com/regenrek/peakypanes/internal/userpath"
)

const (
	paneDirName       = "panes"
	quarantineDirName = "quarantine"
	snapshotExt       = ".json.gz"
)

// Store persists pane snapshots to disk.
type Store struct {
	baseDir string
	paneDir string
	cfg     Config

	mu    sync.RWMutex
	panes map[string]PaneSnapshot
}

// NewStore creates a new restore store rooted at cfg.BaseDir.
func NewStore(cfg Config) (*Store, error) {
	cfg = cfg.Normalized()
	base := strings.TrimSpace(cfg.BaseDir)
	if base == "" {
		return nil, errors.New("sessionrestore: base dir is required")
	}
	base = filepath.Clean(userpath.ExpandUser(base))
	paneDir := filepath.Join(base, paneDirName)
	if err := os.MkdirAll(paneDir, 0o700); err != nil {
		return nil, fmt.Errorf("sessionrestore: create pane dir: %w", err)
	}
	return &Store{
		baseDir: base,
		paneDir: paneDir,
		cfg:     cfg,
		panes:   make(map[string]PaneSnapshot),
	}, nil
}

// Load reads persisted snapshots from disk.
func (s *Store) Load(ctx context.Context) error {
	if s == nil {
		return nil
	}
	entries, err := os.ReadDir(s.paneDir)
	if err != nil {
		return fmt.Errorf("sessionrestore: read pane dir: %w", err)
	}
	loaded := make(map[string]PaneSnapshot)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, snapshotExt) {
			continue
		}
		path := filepath.Join(s.paneDir, name)
		snap, err := s.loadSnapshot(path)
		if err != nil {
			s.quarantine(path)
			continue
		}
		if snap.PaneID == "" {
			s.quarantine(path)
			continue
		}
		loaded[snap.PaneID] = snap
	}
	s.mu.Lock()
	s.panes = loaded
	s.mu.Unlock()
	return nil
}

// Snapshot returns the persisted snapshot for a pane.
func (s *Store) Snapshot(paneID string) (PaneSnapshot, bool) {
	if s == nil {
		return PaneSnapshot{}, false
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return PaneSnapshot{}, false
	}
	s.mu.RLock()
	snap, ok := s.panes[paneID]
	s.mu.RUnlock()
	return snap, ok
}

// Snapshots returns all persisted snapshots.
func (s *Store) Snapshots() []PaneSnapshot {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]PaneSnapshot, 0, len(s.panes))
	for _, snap := range s.panes {
		out = append(out, snap)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].CapturedAt.Before(out[j].CapturedAt)
	})
	return out
}

// Save persists a pane snapshot to disk.
func (s *Store) Save(ctx context.Context, snap PaneSnapshot) error {
	if s == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	paneID, err := sanitizeID(snap.PaneID)
	if err != nil {
		return err
	}
	snap.SchemaVersion = CurrentSchemaVersion
	if snap.CapturedAt.IsZero() {
		snap.CapturedAt = time.Now().UTC()
	}
	if snap.Terminal.Cols < 0 {
		snap.Terminal.Cols = 0
	}
	if snap.Terminal.Rows < 0 {
		snap.Terminal.Rows = 0
	}
	data, err := encodeSnapshot(&snap)
	if err != nil {
		return err
	}
	path := filepath.Join(s.paneDir, paneID+snapshotExt)
	if err := atomicfile.Save(path, data, 0o600); err != nil {
		return err
	}
	s.mu.Lock()
	s.panes[paneID] = snap
	s.mu.Unlock()
	return nil
}

// Delete removes a pane snapshot from disk.
func (s *Store) Delete(paneID string) {
	if s == nil {
		return
	}
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return
	}
	path := filepath.Join(s.paneDir, paneID+snapshotExt)
	_ = os.Remove(path)
	s.mu.Lock()
	delete(s.panes, paneID)
	s.mu.Unlock()
}

// GC enforces TTL and disk size caps.
func (s *Store) GC(ctx context.Context, live map[string]struct{}) error {
	if s == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.cfg.TTLInactive > 0 {
		s.expireByTTL(time.Now().UTC())
	}
	if s.cfg.MaxDiskBytes > 0 {
		return s.enforceDiskCap(ctx, live)
	}
	return nil
}

func (s *Store) expireByTTL(now time.Time) {
	if s == nil {
		return
	}
	cutoff := now.Add(-s.cfg.TTLInactive)
	var expired []string
	s.mu.RLock()
	for id, snap := range s.panes {
		last := snap.PaneLastAct
		if snap.CapturedAt.After(last) {
			last = snap.CapturedAt
		}
		if last.Before(cutoff) {
			expired = append(expired, id)
		}
	}
	s.mu.RUnlock()
	for _, id := range expired {
		s.Delete(id)
	}
}

type paneFile struct {
	id         string
	path       string
	size       int64
	lastActive time.Time
	capturedAt time.Time
	live       bool
}

func (s *Store) enforceDiskCap(ctx context.Context, live map[string]struct{}) error {
	if s == nil {
		return nil
	}
	entries, err := os.ReadDir(s.paneDir)
	if err != nil {
		return fmt.Errorf("sessionrestore: read pane dir: %w", err)
	}
	files, total, err := s.collectPaneFiles(ctx, entries, live)
	if err != nil {
		return err
	}
	if total <= s.cfg.MaxDiskBytes {
		return nil
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].live != files[j].live {
			return !files[i].live
		}
		if !files[i].lastActive.Equal(files[j].lastActive) {
			return files[i].lastActive.Before(files[j].lastActive)
		}
		return files[i].capturedAt.Before(files[j].capturedAt)
	})
	return s.evictPaneFiles(ctx, files, total)
}

func (s *Store) collectPaneFiles(ctx context.Context, entries []os.DirEntry, live map[string]struct{}) ([]paneFile, int64, error) {
	var files []paneFile
	total := int64(0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		file, size, ok := s.paneFileFromEntry(entry, live)
		if !ok {
			continue
		}
		total += size
		files = append(files, file)
	}
	return files, total, nil
}

func (s *Store) paneFileFromEntry(entry os.DirEntry, live map[string]struct{}) (paneFile, int64, bool) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), snapshotExt) {
		return paneFile{}, 0, false
	}
	id := strings.TrimSuffix(entry.Name(), snapshotExt)
	info, err := entry.Info()
	if err != nil {
		return paneFile{}, 0, false
	}
	snap, ok := s.panes[id]
	last := time.Time{}
	capturedAt := time.Time{}
	if ok {
		last = snap.PaneLastAct
		if snap.CapturedAt.After(last) {
			last = snap.CapturedAt
		}
		capturedAt = snap.CapturedAt
	}
	if last.IsZero() {
		last = info.ModTime()
	}
	if capturedAt.IsZero() {
		capturedAt = info.ModTime()
	}
	_, livePane := live[id]
	return paneFile{
		id:         id,
		path:       filepath.Join(s.paneDir, entry.Name()),
		size:       info.Size(),
		lastActive: last,
		capturedAt: capturedAt,
		live:       livePane,
	}, info.Size(), true
}

func (s *Store) evictPaneFiles(ctx context.Context, files []paneFile, total int64) error {
	for _, file := range files {
		if total <= s.cfg.MaxDiskBytes {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		_ = os.Remove(file.path)
		total -= file.size
		s.mu.Lock()
		delete(s.panes, file.id)
		s.mu.Unlock()
	}
	return nil
}

func (s *Store) loadSnapshot(path string) (PaneSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PaneSnapshot{}, fmt.Errorf("sessionrestore: read snapshot: %w", err)
	}
	snap, err := decodeSnapshot(bytes.NewReader(data))
	if err != nil {
		return PaneSnapshot{}, err
	}
	if snap.SchemaVersion != CurrentSchemaVersion {
		return PaneSnapshot{}, fmt.Errorf("sessionrestore: unknown schema %d", snap.SchemaVersion)
	}
	return snap, nil
}

func (s *Store) quarantine(path string) {
	_ = os.MkdirAll(filepath.Join(s.baseDir, quarantineDirName), 0o700)
	base := filepath.Base(path)
	now := time.Now().UTC().Format("20060102-150405")
	target := filepath.Join(s.baseDir, quarantineDirName, base+"-"+now)
	_ = os.Rename(path, target)
}

func encodeSnapshot(snap *PaneSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, errors.New("sessionrestore: snapshot is nil")
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gz)
	if err := enc.Encode(snap); err != nil {
		_ = gz.Close()
		return nil, fmt.Errorf("sessionrestore: encode snapshot: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("sessionrestore: close snapshot gzip: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeSnapshot(r io.Reader) (PaneSnapshot, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return PaneSnapshot{}, fmt.Errorf("sessionrestore: open snapshot gzip: %w", err)
	}
	var snap PaneSnapshot
	decErr := json.NewDecoder(gz).Decode(&snap)
	closeErr := gz.Close()
	if decErr != nil {
		if closeErr != nil {
			return PaneSnapshot{}, fmt.Errorf("sessionrestore: decode snapshot: %v (close gzip: %w)", decErr, closeErr)
		}
		return PaneSnapshot{}, fmt.Errorf("sessionrestore: decode snapshot: %w", decErr)
	}
	if closeErr != nil {
		return PaneSnapshot{}, fmt.Errorf("sessionrestore: close snapshot gzip: %w", closeErr)
	}
	return snap, nil
}

func sanitizeID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("sessionrestore: pane id is required")
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("sessionrestore: invalid pane id %q", value)
	}
	return value, nil
}
