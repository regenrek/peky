package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/runenv"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

type runEnvOptions struct {
	freshConfig  bool
	temporaryRun bool
}

type envSnapshot struct {
	key   string
	value string
	ok    bool
}

func applyRunEnvFromFlags(cmd *cli.Command, version string) (func(), error) {
	if cmd == nil {
		return func() {}, nil
	}
	opts := runEnvOptions{
		freshConfig:  cmd.Bool("fresh-config"),
		temporaryRun: cmd.Bool("temporary-run"),
	}
	if opts.temporaryRun {
		opts.freshConfig = true
	}
	return applyRunEnv(opts, version)
}

func applyRunEnv(opts runEnvOptions, version string) (func(), error) {
	if !opts.freshConfig && !opts.temporaryRun {
		return func() {}, nil
	}
	original := captureEnv(runenv.RuntimeDirEnv, runenv.ConfigDirEnv, runenv.FreshConfigEnv)
	cleanup := func() {
		restoreEnv(original)
	}

	if opts.temporaryRun {
		root, err := os.MkdirTemp("", "peakypanes-run-")
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("create temporary run dir: %w", err)
		}
		runtimeDir := filepath.Join(root, "runtime")
		configDir := filepath.Join(root, "config")
		layoutsDir := filepath.Join(configDir, "layouts")
		if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
			_ = os.RemoveAll(root)
			cleanup()
			return nil, fmt.Errorf("create temporary runtime dir: %w", err)
		}
		if err := os.MkdirAll(layoutsDir, 0o755); err != nil {
			_ = os.RemoveAll(root)
			cleanup()
			return nil, fmt.Errorf("create temporary config dir: %w", err)
		}
		if err := os.Setenv(runenv.RuntimeDirEnv, runtimeDir); err != nil {
			_ = os.RemoveAll(root)
			cleanup()
			return nil, fmt.Errorf("set runtime dir: %w", err)
		}
		if err := os.Setenv(runenv.ConfigDirEnv, configDir); err != nil {
			_ = os.RemoveAll(root)
			cleanup()
			return nil, fmt.Errorf("set config dir: %w", err)
		}
		if err := os.Setenv(runenv.FreshConfigEnv, "1"); err != nil {
			_ = os.RemoveAll(root)
			cleanup()
			return nil, fmt.Errorf("set fresh config: %w", err)
		}

		return func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = sessiond.StopDaemon(ctx, version)
			cancel()
			_ = os.RemoveAll(root)
			restoreEnv(original)
		}, nil
	}

	if err := os.Setenv(runenv.FreshConfigEnv, "1"); err != nil {
		cleanup()
		return nil, fmt.Errorf("set fresh config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	running, err := sessiond.DaemonRunning(ctx, version)
	cancel()
	if err != nil {
		cleanup()
		return nil, err
	}
	if running {
		cleanup()
		return nil, fmt.Errorf("daemon already running; stop it or use --temporary-run")
	}
	return cleanup, nil
}

func captureEnv(keys ...string) []envSnapshot {
	snaps := make([]envSnapshot, 0, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		snaps = append(snaps, envSnapshot{key: key, value: value, ok: ok})
	}
	return snaps
}

func restoreEnv(snaps []envSnapshot) {
	for _, snap := range snaps {
		if snap.ok {
			_ = os.Setenv(snap.key, snap.value)
		} else {
			_ = os.Unsetenv(snap.key)
		}
	}
}
