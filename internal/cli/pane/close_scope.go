package pane

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func closePaneScope(ctx root.CommandContext, client *sessiond.Client, scope string, start time.Time, meta output.Meta) error {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Errorf("scope is required")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	snap, err := client.SnapshotState(ctxTimeout, 0)
	cancel()
	if err != nil {
		return err
	}
	targets, err := sessiond.ResolveScopeTargets(scope, snap.Sessions, snap.FocusedSession, snap.FocusedPaneID)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no panes found for scope %q", scope)
	}

	results := make([]output.TargetResult, 0, len(targets))
	failures := 0
	for _, paneID := range targets {
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
		err := client.ClosePaneByID(ctxTimeout, paneID)
		cancel()
		if err != nil {
			failures++
			results = append(results, output.TargetResult{
				Target:  output.TargetRef{Type: "pane", ID: paneID},
				Status:  "failed",
				Message: err.Error(),
			})
			continue
		}
		results = append(results, output.TargetResult{
			Target: output.TargetRef{Type: "pane", ID: paneID},
			Status: "ok",
		})
	}

	status := "ok"
	if failures > 0 {
		status = "partial"
	}

	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "pane.close",
			Status:  status,
			Results: results,
			Details: map[string]any{"scope": scope, "count": len(targets)},
		})
	}
	if failures > 0 {
		return fmt.Errorf("closed %d panes; %d failed", len(targets), failures)
	}
	return writef(ctx.Out, "Closed %d panes (%s)\n", len(targets), scope)
}
