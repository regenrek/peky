package devwatch

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"
)

// Run starts the dev watcher loop.
func Run(ctx context.Context, cfg Config) error {
	prepared, err := prepareConfig(cfg)
	if err != nil {
		return err
	}
	watcher := newPollWatcher(prepared)
	changes := watcher.Changes(ctx.Done())

	for {
		if err := runBuild(ctx, prepared); err != nil {
			prepared.Logger.Printf("build failed: %v", err)
			if !waitForChange(ctx, changes) {
				return err
			}
			continue
		}
		restart, err := runWithRestart(ctx, prepared, changes)
		if err != nil {
			return err
		}
		if !restart {
			return nil
		}
	}
}

func runBuild(ctx context.Context, cfg Config) error {
	cmd := exec.CommandContext(ctx, cfg.BuildCmd[0], cfg.BuildCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runWithRestart(ctx context.Context, cfg Config, changes <-chan struct{}) (bool, error) {
	cmd := exec.Command(cfg.RunCmd[0], cfg.RunCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return false, err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			requestStop(cmd, cfg.ShutdownTimeout, done)
			return false, ctx.Err()
		case <-changes:
			requestStop(cmd, cfg.ShutdownTimeout, done)
			if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
				cfg.Logger.Printf("process exit: %v", err)
			}
			drainChanges(changes)
			return true, nil
		case err := <-done:
			if err != nil {
				return false, err
			}
			return false, nil
		}
	}
}

func waitForChange(ctx context.Context, changes <-chan struct{}) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case <-changes:
			drainChanges(changes)
			return true
		}
	}
}

func requestStop(cmd *exec.Cmd, timeout time.Duration, done <-chan error) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	if timeout <= 0 {
		return
	}
	select {
	case <-done:
		return
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
	}
}

func drainChanges(changes <-chan struct{}) {
	for {
		select {
		case <-changes:
		default:
			return
		}
	}
}
