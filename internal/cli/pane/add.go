package pane

import (
	"context"
	"fmt"
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

	sessionName := strings.TrimSpace(ctx.Cmd.String("session"))
	paneIndex := intFlagString(ctx.Cmd, "index")
	paneID := strings.TrimSpace(ctx.Cmd.String("pane-id"))
	orientation := strings.ToLower(strings.TrimSpace(ctx.Cmd.String("orientation")))
	if orientation == "" {
		orientation = "vertical"
	}
	if orientation != "vertical" && orientation != "horizontal" {
		return fmt.Errorf("invalid orientation %q (allowed: vertical, horizontal)", orientation)
	}
	vertical := orientation == "vertical"
	percent := ctx.Cmd.Int("percent")

	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()

	targetSession, targetIndex, err := resolveAddTarget(ctxTimeout, client, sessionName, paneIndex, paneID)
	if err != nil {
		return err
	}
	newIndex, err := client.SplitPane(ctxTimeout, targetSession, targetIndex, vertical, percent)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "pane.add",
			Status: "ok",
			Details: map[string]any{
				"session":     targetSession,
				"split_from":  targetIndex,
				"new_index":   newIndex,
				"orientation": orientation,
			},
		})
	}
	return writef(ctx.Out, "Pane added %s\n", newIndex)
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

func findPaneByID(sessions []native.SessionSnapshot, paneID string) (string, string, bool) {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return "", "", false
	}
	for _, session := range sessions {
		for _, pane := range session.Panes {
			if pane.ID == paneID {
				return session.Name, pane.Index, true
			}
		}
	}
	return "", "", false
}
