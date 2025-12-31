package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers daemon handlers.
func Register(reg *root.Registry) {
	reg.Register("daemon", runDaemon)
	reg.Register("daemon.restart", runRestart)
}

func runDaemon(ctx root.CommandContext) error {
	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:       ctx.Deps.Version,
		StateDebounce: sessiond.DefaultStateDebounce,
		HandleSignals: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	if err := daemon.Run(); err != nil {
		return fmt.Errorf("daemon failed: %w", err)
	}
	return nil
}

func runRestart(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("daemon.restart", ctx.Deps.Version)
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, 15*time.Second)
	defer cancel()
	if err := sessiond.RestartDaemon(ctxTimeout, ctx.Deps.Version); err != nil {
		return fmt.Errorf("failed to restart daemon: %w", err)
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "daemon.restart",
			Status: "ok",
		})
	}
	if _, err := fmt.Fprintln(ctx.ErrOut, "Daemon restarted."); err != nil {
		return err
	}
	return nil
}
