package pane

import (
	"context"
	"fmt"
	"strings"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

type layoutOutputMode int

const (
	layoutOutputNone layoutOutputMode = iota
	layoutOutputAfter
	layoutOutputDiff
)

func parseLayoutOutputMode(ctx root.CommandContext) (layoutOutputMode, error) {
	if ctx.Cmd == nil {
		return layoutOutputNone, nil
	}
	after := ctx.Cmd.Bool("after")
	diff := ctx.Cmd.Bool("diff")
	if after || diff {
		if !ctx.JSON {
			return layoutOutputNone, fmt.Errorf("--after/--diff require --json")
		}
	}
	if diff {
		return layoutOutputDiff, nil
	}
	if after {
		return layoutOutputAfter, nil
	}
	return layoutOutputNone, nil
}

func buildLayoutState(sessionName, paneID string, opResp *sessiond.LayoutOpResponse, before, after *layout.TreeSnapshot) *output.LayoutState {
	state := &output.LayoutState{
		Session: strings.TrimSpace(sessionName),
		PaneID:  strings.TrimSpace(paneID),
		Before:  before,
		After:   after,
	}
	if state.Session == "" {
		state.Session = sessionName
	}
	if opResp == nil {
		return state
	}
	changed := opResp.Changed
	snapped := opResp.Snapped
	state.Changed = &changed
	state.Snapped = &snapped
	state.SnapState = &output.LayoutSnapState{
		Active: opResp.SnapState.Active,
		Target: opResp.SnapState.Target,
	}
	if len(opResp.Affected) > 0 {
		state.Affected = append([]string(nil), opResp.Affected...)
	}
	return state
}

func captureLayoutBefore(ctx context.Context, client *sessiond.Client, mode layoutOutputMode, sessionName string) (*layout.TreeSnapshot, error) {
	if mode != layoutOutputDiff {
		return nil, nil
	}
	return captureLayoutSnapshot(ctx, client, sessionName, true)
}

func captureLayoutAfter(ctx context.Context, client *sessiond.Client, mode layoutOutputMode, sessionName string) (*layout.TreeSnapshot, error) {
	if mode == layoutOutputNone {
		return nil, nil
	}
	return captureLayoutSnapshot(ctx, client, sessionName, false)
}

func captureLayoutSnapshot(ctx context.Context, client *sessiond.Client, sessionName string, require bool) (*layout.TreeSnapshot, error) {
	if client == nil {
		return nil, fmt.Errorf("sessiond client unavailable")
	}
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		if require {
			return nil, fmt.Errorf("session name is required")
		}
		return nil, nil
	}
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		return nil, err
	}
	snap, ok := findLayoutSnapshot(resp.Sessions, sessionName)
	if !ok {
		if require {
			return nil, fmt.Errorf("session %q not found", sessionName)
		}
		return nil, nil
	}
	return snap, nil
}

func findLayoutSnapshot(sessions []native.SessionSnapshot, sessionName string) (*layout.TreeSnapshot, bool) {
	sessionName = strings.TrimSpace(sessionName)
	for _, session := range sessions {
		if strings.TrimSpace(session.Name) == sessionName {
			return session.LayoutTree, true
		}
	}
	return nil, false
}
