package sessiond

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/native"
)

func (d *Daemon) resolveScopeTargets(scope string) ([]string, error) {
	if d == nil {
		return nil, errors.New("sessiond: daemon unavailable")
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil, errors.New("sessiond: scope is required")
	}
	if d.manager == nil {
		return nil, errors.New("sessiond: manager unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	sessions := d.manager.Snapshot(ctx, 0)
	if len(sessions) == 0 {
		return nil, errors.New("sessiond: no sessions available")
	}
	switch strings.ToLower(scope) {
	case "all":
		return collectPaneIDs(sessions), nil
	case "session":
		name := d.resolveFocusedSession(sessions)
		if name == "" {
			return nil, errors.New("sessiond: focused session unavailable")
		}
		return collectSessionPaneIDs(sessions, name), nil
	case "project":
		path := d.resolveFocusedProjectPath(sessions)
		if path == "" {
			return nil, errors.New("sessiond: focused project unavailable")
		}
		return collectProjectPaneIDs(sessions, path), nil
	default:
		return nil, fmt.Errorf("sessiond: unknown scope %q", scope)
	}
}

func collectPaneIDs(sessions []native.SessionSnapshot) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(sessions)*2)
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == "" {
				continue
			}
			if _, ok := seen[pane.ID]; ok {
				continue
			}
			seen[pane.ID] = struct{}{}
			out = append(out, pane.ID)
		}
	}
	return out
}

func collectSessionPaneIDs(sessions []native.SessionSnapshot, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for _, session := range sessions {
		if session.Name != name {
			continue
		}
		ids := make([]string, 0, len(session.Panes))
		for _, pane := range session.Panes {
			if pane.ID != "" {
				ids = append(ids, pane.ID)
			}
		}
		return ids
	}
	return nil
}

func collectProjectPaneIDs(sessions []native.SessionSnapshot, path string) []string {
	path = normalizeProjectPath(path)
	if path == "" {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(sessions)*2)
	for _, session := range sessions {
		if normalizeProjectPath(session.Path) != path {
			continue
		}
		for _, pane := range session.Panes {
			if pane.ID == "" {
				continue
			}
			if _, ok := seen[pane.ID]; ok {
				continue
			}
			seen[pane.ID] = struct{}{}
			out = append(out, pane.ID)
		}
	}
	return out
}

func (d *Daemon) resolveFocusedSession(sessions []native.SessionSnapshot) string {
	session, pane := d.focusState()
	if session != "" {
		return session
	}
	if pane != "" {
		for _, snap := range sessions {
			for _, paneSnap := range snap.Panes {
				if paneSnap.ID == pane {
					return snap.Name
				}
			}
		}
	}
	if len(sessions) == 1 {
		return sessions[0].Name
	}
	return ""
}

func (d *Daemon) resolveFocusedProjectPath(sessions []native.SessionSnapshot) string {
	sessionName := d.resolveFocusedSession(sessions)
	if sessionName != "" {
		for _, snap := range sessions {
			if snap.Name == sessionName {
				return normalizeProjectPath(snap.Path)
			}
		}
	}
	paths := make(map[string]struct{})
	for _, snap := range sessions {
		path := normalizeProjectPath(snap.Path)
		if path == "" {
			continue
		}
		paths[path] = struct{}{}
	}
	if len(paths) == 1 {
		for path := range paths {
			return path
		}
	}
	return ""
}

func normalizeProjectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
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
