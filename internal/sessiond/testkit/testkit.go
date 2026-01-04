package testkit

import (
	"context"
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// WaitForSessionSnapshot waits for a session snapshot using SnapshotState and daemon events.
func WaitForSessionSnapshot(ctx context.Context, client *sessiond.Client, name string) (native.SessionSnapshot, error) {
	if client == nil {
		return native.SessionSnapshot{}, fmt.Errorf("session client unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return native.SessionSnapshot{}, fmt.Errorf("session name required")
	}
	if snap, ok, err := snapshotByName(ctx, client, name); err != nil {
		return native.SessionSnapshot{}, err
	} else if ok {
		return snap, nil
	}
	return waitForSessionSnapshotEvents(ctx, client, name)
}

func waitForSessionSnapshotEvents(ctx context.Context, client *sessiond.Client, name string) (native.SessionSnapshot, error) {
	events := client.Events()
	if events == nil {
		return native.SessionSnapshot{}, fmt.Errorf("session events unavailable")
	}
	for {
		select {
		case <-ctx.Done():
			return native.SessionSnapshot{}, fmt.Errorf("session snapshot missing for %q: %w", name, ctx.Err())
		case evt, ok := <-events:
			if !ok {
				return native.SessionSnapshot{}, fmt.Errorf("session event channel closed while waiting for %q", name)
			}
			if !shouldCheckSnapshotOnEvent(evt.Type) {
				continue
			}
			snap, ok, err := snapshotByName(ctx, client, name)
			if err != nil {
				return native.SessionSnapshot{}, err
			}
			if ok {
				return snap, nil
			}
		}
	}
}

func shouldCheckSnapshotOnEvent(eventType sessiond.EventType) bool {
	switch eventType {
	case sessiond.EventSessionChanged, sessiond.EventPaneUpdated, sessiond.EventPaneMetaChanged, sessiond.EventFocus:
		return true
	default:
		return false
	}
}

func snapshotByName(ctx context.Context, client *sessiond.Client, name string) (native.SessionSnapshot, bool, error) {
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		return native.SessionSnapshot{}, false, err
	}
	for _, snap := range resp.Sessions {
		if snap.Name == name {
			return snap, true, nil
		}
	}
	return native.SessionSnapshot{}, false, nil
}
