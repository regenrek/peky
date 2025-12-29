package devwatch

import (
	"errors"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type pollWatcher struct {
	watchPaths []string
	extensions map[string]struct{}
	interval   time.Duration
	debounce   time.Duration
	logger     logger
}

type logger interface {
	Printf(format string, args ...any)
}

func newPollWatcher(cfg Config) *pollWatcher {
	ext := make(map[string]struct{}, len(cfg.Extensions))
	for _, value := range cfg.Extensions {
		ext[value] = struct{}{}
	}
	return &pollWatcher{
		watchPaths: append([]string{}, cfg.WatchPaths...),
		extensions: ext,
		interval:   cfg.Interval,
		debounce:   cfg.Debounce,
		logger:     cfg.Logger,
	}
}

func (w *pollWatcher) Changes(ctxDone <-chan struct{}) <-chan struct{} {
	changes := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		fingerprint, err := w.scanFingerprint()
		if err != nil {
			w.logger.Printf("initial scan failed: %v", err)
		}
		var (
			pending  bool
			debounce *time.Timer
		)
		for {
			select {
			case <-ctxDone:
				return
			case <-ticker.C:
				updated, err := w.scanFingerprint()
				if err != nil {
					w.logger.Printf("scan failed: %v", err)
					continue
				}
				if updated == fingerprint {
					continue
				}
				fingerprint = updated
				if w.debounce <= 0 {
					w.signalChange(changes)
					continue
				}
				pending = true
				if debounce == nil {
					debounce = time.NewTimer(w.debounce)
					continue
				}
				if !debounce.Stop() {
					select {
					case <-debounce.C:
					default:
					}
				}
				debounce.Reset(w.debounce)
			case <-timerChan(debounce):
				if pending {
					pending = false
					w.signalChange(changes)
				}
			}
		}
	}()
	return changes
}

func timerChan(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

func (w *pollWatcher) signalChange(ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (w *pollWatcher) scanFingerprint() (uint64, error) {
	files, err := w.collectFiles()
	if err != nil {
		return 0, err
	}
	sort.Strings(files)
	h := fnv.New64a()
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		_, _ = h.Write([]byte(path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(info.ModTime().UTC().Format(time.RFC3339Nano)))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(strconvInt64(info.Size())))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64(), nil
}

func (w *pollWatcher) collectFiles() ([]string, error) {
	files := make([]string, 0, 256)
	var scanErr error
	for _, root := range w.watchPaths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				w.logger.Printf("watch path missing: %s", root)
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			if w.includeFile(root, true) {
				files = append(files, root)
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if shouldSkipDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if w.includeFile(path, false) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			scanErr = err
		}
	}
	if scanErr != nil {
		return files, scanErr
	}
	return files, nil
}

func shouldSkipDir(name string) bool {
	if name == ".git" || name == "node_modules" || name == "dist" || name == "vendor" {
		return true
	}
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}
	return false
}

func (w *pollWatcher) includeFile(path string, explicit bool) bool {
	if explicit {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := w.extensions[ext]
	return ok
}

func strconvInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
