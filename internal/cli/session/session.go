package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/cli/transform"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/sessionpolicy"
)

// Register registers session handlers.
func Register(reg *root.Registry) {
	reg.Register("session.list", runList)
	reg.Register("session.start", runStart)
	reg.Register("session.close", runClose)
	reg.Register("session.rename", runRename)
	reg.Register("session.focus", runFocus)
	reg.Register("session.snapshot", runSnapshot)
}

func runList(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("session.list", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.SnapshotState(ctxTimeout, 0)
	if err != nil {
		return err
	}
	sessions := transform.SessionSummaries(resp.Sessions)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Sessions []output.SessionSummary `json:"sessions"`
			Total    int                     `json:"total"`
		}{Sessions: sessions, Total: len(sessions)})
	}
	for _, session := range sessions {
		if _, err := fmt.Fprintln(ctx.Out, session.Name); err != nil {
			return err
		}
	}
	return nil
}

func runStart(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("session.start", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	path := strings.TrimSpace(ctx.Cmd.String("path"))
	if path == "" {
		cwd, err := root.ResolveWorkDir(ctx)
		if err != nil {
			return err
		}
		path = cwd
	}
	env, err := sessionpolicy.ValidateEnvList(ctx.Cmd.StringSlice("env"))
	if err != nil {
		return err
	}
	paneCount, err := sessionpolicy.ValidatePaneCount(ctx.Cmd.Int("panes"))
	if err != nil {
		return err
	}
	req := sessiond.StartSessionRequest{
		Name:       strings.TrimSpace(ctx.Cmd.String("name")),
		Path:       path,
		LayoutName: strings.TrimSpace(ctx.Cmd.String("layout")),
		PaneCount:  paneCount,
		Env:        env,
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.StartSession(ctxTimeout, req)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "session.start",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "session", ID: resp.Name}},
			Details: map[string]any{
				"name":   resp.Name,
				"path":   resp.Path,
				"layout": resp.LayoutName,
			},
		})
	}
	if _, err := fmt.Fprintf(ctx.Out, "Started session %s\n", resp.Name); err != nil {
		return err
	}
	return nil
}

func runClose(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("session.close", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	name := strings.TrimSpace(ctx.Cmd.String("name"))
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.KillSession(ctxTimeout, name); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "session.close",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "session", ID: name}},
		})
	}
	if _, err := fmt.Fprintf(ctx.Out, "Closed session %s\n", name); err != nil {
		return err
	}
	return nil
}

func runRename(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("session.rename", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	oldName := strings.TrimSpace(ctx.Cmd.String("old"))
	newName := strings.TrimSpace(ctx.Cmd.String("new"))
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.RenameSession(ctxTimeout, oldName, newName)
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "session.rename",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "session", ID: resp.NewName}},
			Details: map[string]any{"name": resp.NewName},
		})
	}
	if _, err := fmt.Fprintf(ctx.Out, "Renamed session to %s\n", resp.NewName); err != nil {
		return err
	}
	return nil
}

func runFocus(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("session.focus", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	name := strings.TrimSpace(ctx.Cmd.String("name"))
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	if err := client.FocusSession(ctxTimeout, name); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "session.focus",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "session", ID: name}},
		})
	}
	if _, err := fmt.Fprintf(ctx.Out, "Focused session %s\n", name); err != nil {
		return err
	}
	return nil
}

func runSnapshot(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("session.snapshot", ctx.Deps.Version)
	client, cleanup, err := connect(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	defer cancel()
	resp, err := client.SnapshotState(ctxTimeout, 0)
	if err != nil {
		return err
	}
	ws, err := transform.LoadWorkspace()
	if err != nil {
		return err
	}
	snapshot := transform.BuildSnapshot(resp.Sessions, ws, resp.FocusedSession, resp.FocusedPaneID)
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, struct {
			Snapshot output.Snapshot `json:"snapshot"`
		}{Snapshot: snapshot})
	}
	for _, project := range snapshot.Projects {
		for _, session := range project.Sessions {
			if _, err := fmt.Fprintln(ctx.Out, session.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func connect(ctx root.CommandContext) (*sessiond.Client, func(), error) {
	connect := ctx.Deps.Connect
	if connect == nil {
		return nil, func() {}, fmt.Errorf("daemon connection not configured")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, commandTimeout(ctx))
	client, err := connect(ctxTimeout, ctx.Deps.Version)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	cleanup := func() {
		cancel()
		_ = client.Close()
	}
	return client, cleanup, nil
}

func commandTimeout(ctx root.CommandContext) time.Duration {
	if ctx.Cmd.IsSet("timeout") {
		return ctx.Cmd.Duration("timeout")
	}
	return 10 * time.Second
}
