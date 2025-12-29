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
	if d == nil || d.manager == nil {
		return nil
	}
	path := strings.TrimSpace(d.statePath)
	if path == "" {
		return nil
	}
	st, err := state.Load(path)
	if err != nil {
		if errors.Is(err, state.ErrUnknownSchema) {
			log.Printf("sessiond: ignoring state with unknown schema")
			return nil
		}
		return err
	}
	if st == nil || len(st.Sessions) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, name := range d.manager.SessionNames() {
		seen[name] = struct{}{}
	}

	for _, session := range st.Sessions {
		name, err := validateSessionName(session.Name)
		if err != nil {
			log.Printf("sessiond: restore skipped invalid session %q: %v", session.Name, err)
			continue
		}
		name = uniqueSessionName(name, seen)
		seen[name] = struct{}{}

		path := strings.TrimSpace(session.Path)
		if path != "" {
			if _, err := validatePath(path); err != nil {
				log.Printf("sessiond: restore session %q invalid path %q: %v", name, path, err)
				path = ""
			}
		}

		paneSpecs := make([]native.PaneRestoreSpec, len(session.Panes))
		activeIndex := strings.TrimSpace(session.ActivePaneIndex)
		activeSet := false
		for i, pane := range session.Panes {
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

		spec := native.SessionRestoreSpec{
			Name:       name,
			Path:       path,
			LayoutName: session.LayoutName,
			CreatedAt:  session.CreatedAt,
			Env:        append([]string(nil), session.Env...),
			Panes:      paneSpecs,
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultOpTimeout)
		_, err = d.manager.RestoreSession(ctx, spec)
		cancel()
		if err != nil {
			log.Printf("sessiond: restore session %q failed: %v", name, err)
			continue
		}
	}

	d.queuePersistState()
	return nil
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
