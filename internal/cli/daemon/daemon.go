package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/runenv"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers daemon handlers.
func Register(reg *root.Registry) {
	reg.Register("daemon", runDaemon)
	reg.Register("daemon.start", runDaemon)
	reg.Register("daemon.stop", runStop)
	reg.Register("daemon.restart", runRestart)
}

var stopDaemon = sessiond.StopDaemon
var restartDaemon = sessiond.RestartDaemon

const defaultPprofAddr = "127.0.0.1:6060"

func runDaemon(ctx root.CommandContext) error {
	pprofAddr, err := resolvePprofAddr(ctx)
	if err != nil {
		return err
	}
	fresh := runenv.FreshConfigEnabled()
	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:                 ctx.Deps.Version,
		StateDebounce:           sessiond.DefaultStateDebounce,
		HandleSignals:           true,
		SkipRestore:             fresh,
		DisableStatePersistence: fresh,
		PprofAddr:               pprofAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	sigCtx, stop := signal.NotifyContext(ctx.Context, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-sigCtx.Done()
		_ = daemon.Stop()
	}()
	if err := daemon.Run(); err != nil {
		return fmt.Errorf("daemon failed: %w", err)
	}
	return nil
}

func resolvePprofAddr(ctx root.CommandContext) (string, error) {
	addr := strings.TrimSpace(ctx.Cmd.String("pprof-addr"))
	if ctx.Cmd.IsSet("pprof-addr") && addr == "" {
		return "", fmt.Errorf("pprof address is required")
	}
	if ctx.Cmd.Bool("pprof") && addr == "" {
		addr = defaultPprofAddr
	}
	return addr, nil
}

func runStop(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("daemon.stop", ctx.Deps.Version)
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, 15*time.Second)
	defer cancel()
	if err := stopDaemon(ctxTimeout, ctx.Deps.Version); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "daemon.stop",
			Status: "ok",
		})
	}
	if _, err := fmt.Fprintln(ctx.ErrOut, "Daemon stopped."); err != nil {
		return err
	}
	return nil
}

func runRestart(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("daemon.restart", ctx.Deps.Version)
	ctxTimeout, cancel := context.WithTimeout(ctx.Context, 15*time.Second)
	defer cancel()
	if err := restartDaemon(ctxTimeout, ctx.Deps.Version); err != nil {
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
