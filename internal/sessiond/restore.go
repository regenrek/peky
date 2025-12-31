package sessiond

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond/state"
)

func (d *Daemon) restorePersistedState() error {
	if d == nil || d.manager == nil || d.skipRestore {
		return nil
	}
	path := strings.TrimSpace(d.statePath)
	if path == "" {
		return nil
	}
	st, err := loadPersistedState(path)
	if err != nil || st == nil || len(st.Sessions) == 0 {
		return nil
	}

	seen := sessionNameSet(d.manager.SessionNames())
	for _, session := range st.Sessions {
		d.restoreSessionEntry(session, seen)
	}

	d.queuePersistState()
	return nil
}

func loadPersistedState(path string) (*state.RuntimeState, error) {
	st, err := state.Load(path)
	if err != nil {
		if errors.Is(err, state.ErrUnknownSchema) {
			log.Printf("sessiond: ignoring state with unknown schema")
			return nil, nil
		}
		return nil, err
	}
	return st, nil
}

func sessionNameSet(names []string) map[string]struct{} {
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		seen[name] = struct{}{}
	}
	return seen
}

func (d *Daemon) restoreSessionEntry(session state.Session, seen map[string]struct{}) {
	name, ok := restoreSessionName(session.Name, seen)
	if !ok {
		return
	}
	seen[name] = struct{}{}

	path := restoreSessionPath(name, session.Path)
	paneSpecs := buildPaneRestoreSpecs(session.Panes, session.ActivePaneIndex)

	spec := native.SessionRestoreSpec{
		Name:       name,
		Path:       path,
		LayoutName: session.LayoutName,
		CreatedAt:  session.CreatedAt,
		Env:        append([]string(nil), session.Env...),
		Panes:      paneSpecs,
	}

	if err := d.restoreSessionSpec(spec); err != nil {
		log.Printf("sessiond: restore session %q failed: %v", name, err)
	}
}

func restoreSessionName(raw string, seen map[string]struct{}) (string, bool) {
	name, err := validateSessionName(raw)
	if err != nil {
		log.Printf("sessiond: restore skipped invalid session %q: %v", raw, err)
		return "", false
	}
	name = uniqueSessionName(name, seen)
	return name, true
}

func restoreSessionPath(name, raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		return ""
	}
	if _, err := validatePath(path); err != nil {
		log.Printf("sessiond: restore session %q invalid path %q: %v", name, path, err)
		return ""
	}
	return path
}

func buildPaneRestoreSpecs(panes []state.Pane, activeIndex string) []native.PaneRestoreSpec {
	paneSpecs := make([]native.PaneRestoreSpec, len(panes))
	activeIndex = strings.TrimSpace(activeIndex)
	activeSet := false
	for i, pane := range panes {
		active := pane.Index == activeIndex && activeIndex != ""
		if active {
			activeSet = true
		}
		paneSpecs[i] = native.PaneRestoreSpec{
			Index:         pane.Index,
			Title:         pane.Title,
			Command:       pane.Command,
			StartCommand:  pane.StartCommand,
			Active:        active,
			Left:          pane.Left,
			Top:           pane.Top,
			Width:         pane.Width,
			Height:        pane.Height,
			RestoreFailed: pane.RestoreFailed,
			RestoreError:  pane.RestoreError,
		}
	}
	if !activeSet && len(paneSpecs) > 0 {
		paneSpecs[0].Active = true
	}
	return paneSpecs
}

func (d *Daemon) restoreSessionSpec(spec native.SessionRestoreSpec) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
	defer cancel()
	_, err := d.manager.RestoreSession(ctx, spec)
	return err
}

func uniqueSessionName(base string, seen map[string]struct{}) string {
	if _, exists := seen[base]; !exists {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s (%d)", base, i)
		if _, exists := seen[candidate]; !exists {
			return candidate
		}
	}
}
