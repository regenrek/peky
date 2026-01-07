package pane

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func runAdd(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("pane.add", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	layoutMode, err := parseLayoutOutputMode(ctx)
	if err != nil {
		return err
	}

	opts, err := parseAddOptions(ctx)
	if err != nil {
		return err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()

	targetSession, targetIndex, err := resolveAddTarget(ctxTimeout, client, opts.sessionName, opts.paneIndex, opts.paneID)
	if err != nil {
		return err
	}

	snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	before, err := captureLayoutBefore(snapCtx, client, layoutMode, targetSession)
	cancel()
	if err != nil {
		return err
	}
	newIndexes, err := splitPaneCount(ctxTimeout, client, targetSession, targetIndex, opts)
	if err != nil {
		return err
	}
	var focusedPaneID string
	if opts.focus && len(newIndexes) > 0 {
		focusedPaneID, err = focusPaneByIndex(ctxTimeout, client, targetSession, newIndexes[len(newIndexes)-1])
		if err != nil {
			return err
		}
	}
	if ctx.JSON {
		snapCtx, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		after, err := captureLayoutAfter(snapCtx, client, layoutMode, targetSession)
		cancel()
		if err != nil {
			return err
		}
		meta = output.WithDuration(meta, start)
		details := paneAddDetails(targetSession, targetIndex, opts.orientation, newIndexes, focusedPaneID)
		result := output.ActionResult{
			Action:  "pane.add",
			Status:  "ok",
			Details: details,
		}
		if layoutMode != layoutOutputNone {
			result.Layout = buildLayoutState(targetSession, "", nil, before, after)
		}
		return output.WriteSuccess(ctx.Out, meta, result)
	}
	return writePaneAddOutput(ctx.Out, newIndexes)
}

type addOptions struct {
	sessionName string
	paneIndex   string
	paneID      string
	orientation string
	vertical    bool
	percent     int
	count       int
	focus       bool
}

func parseAddOptions(ctx root.CommandContext) (addOptions, error) {
	orientation := strings.ToLower(strings.TrimSpace(ctx.Cmd.String("orientation")))
	if orientation == "" {
		orientation = "vertical"
	}
	if orientation != "vertical" && orientation != "horizontal" {
		return addOptions{}, fmt.Errorf("invalid orientation %q (allowed: vertical, horizontal)", orientation)
	}
	count := ctx.Cmd.Int("count")
	if count == 0 {
		count = 1
	}
	if count < 1 {
		return addOptions{}, fmt.Errorf("count must be positive")
	}
	return addOptions{
		sessionName: strings.TrimSpace(ctx.Cmd.String("session")),
		paneIndex:   intFlagString(ctx.Cmd, "index"),
		paneID:      strings.TrimSpace(ctx.Cmd.String("pane-id")),
		orientation: orientation,
		vertical:    orientation == "vertical",
		percent:     ctx.Cmd.Int("percent"),
		count:       count,
		focus:       ctx.Cmd.Bool("focus"),
	}, nil
}

func splitPaneCount(ctx context.Context, client *sessiond.Client, sessionName, paneIndex string, opts addOptions) ([]string, error) {
	newIndexes := make([]string, 0, opts.count)
	currentIndex := paneIndex
	for i := 0; i < opts.count; i++ {
		newIndex, err := client.SplitPane(ctx, sessionName, currentIndex, opts.vertical, opts.percent)
		if err != nil {
			return nil, err
		}
		newIndexes = append(newIndexes, newIndex)
		currentIndex = newIndex
	}
	return newIndexes, nil
}

func paneAddDetails(sessionName, paneIndex, orientation string, newIndexes []string, focusedPaneID string) map[string]any {
	details := map[string]any{
		"session":     sessionName,
		"split_from":  paneIndex,
		"orientation": orientation,
	}
	if len(newIndexes) == 1 {
		details["new_index"] = newIndexes[0]
	} else {
		details["new_indexes"] = newIndexes
		details["count"] = len(newIndexes)
	}
	if focusedPaneID != "" {
		details["focused_pane_id"] = focusedPaneID
	}
	return details
}

func writePaneAddOutput(out io.Writer, newIndexes []string) error {
	if len(newIndexes) == 1 {
		return writef(out, "Pane added %s\n", newIndexes[0])
	}
	lastIndex := newIndexes[len(newIndexes)-1]
	return writef(out, "Added %d panes (last index %s)\n", len(newIndexes), lastIndex)
}

func resolveAddTarget(ctx context.Context, client *sessiond.Client, sessionName, paneIndex, paneID string) (string, string, error) {
	if paneID != "" {
		if sessionName != "" || paneIndex != "" {
			return "", "", fmt.Errorf("pane-id cannot be combined with session or index")
		}
		resp, err := client.SnapshotState(ctx, 0)
		if err != nil {
			return "", "", err
		}
		paneID, err = resolvePaneIDFromSnapshot(paneID, resp)
		if err != nil {
			return "", "", err
		}
		targetSession, targetIndex, ok := findPaneByID(resp.Sessions, paneID)
		if !ok {
			return "", "", fmt.Errorf("pane id %q not found", paneID)
		}
		return targetSession, targetIndex, nil
	}
	if sessionName != "" && paneIndex != "" {
		return sessionName, paneIndex, nil
	}
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		return "", "", err
	}
	if sessionName == "" {
		sessionName = strings.TrimSpace(resp.FocusedSession)
		if sessionName == "" {
			return "", "", fmt.Errorf("focused session unavailable; run session focus first")
		}
	}
	if paneIndex == "" {
		idx, err := activePaneIndex(resp.Sessions, sessionName)
		if err != nil {
			return "", "", err
		}
		paneIndex = idx
	}
	return sessionName, paneIndex, nil
}

func activePaneIndex(sessions []native.SessionSnapshot, sessionName string) (string, error) {
	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}
		for _, pane := range session.Panes {
			if pane.Active {
				return pane.Index, nil
			}
		}
		if len(session.Panes) > 0 {
			return session.Panes[0].Index, nil
		}
		return "", fmt.Errorf("session %q has no panes", sessionName)
	}
	return "", fmt.Errorf("session %q not found", sessionName)
}

func focusPaneByIndex(ctx context.Context, client *sessiond.Client, sessionName, paneIndex string) (string, error) {
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		return "", err
	}
	paneID, ok := findPaneIDByIndex(resp.Sessions, sessionName, paneIndex)
	if !ok {
		return "", fmt.Errorf("pane index %q not found in session %q", paneIndex, sessionName)
	}
	if err := client.FocusPane(ctx, paneID); err != nil {
		return "", err
	}
	return paneID, nil
}

func findPaneIDByIndex(sessions []native.SessionSnapshot, sessionName, paneIndex string) (string, bool) {
	sessionName = strings.TrimSpace(sessionName)
	paneIndex = strings.TrimSpace(paneIndex)
	if sessionName == "" || paneIndex == "" {
		return "", false
	}
	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}
		for _, pane := range session.Panes {
			if pane.Index == paneIndex && strings.TrimSpace(pane.ID) != "" {
				return pane.ID, true
			}
		}
	}
	return "", false
}
