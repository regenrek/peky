package mux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/regenrek/peakypanes/internal/tmuxctl"
)

func insideTmux() bool {
	return os.Getenv("TMUX") != "" || os.Getenv("TMUX_PANE") != ""
}

func openDashboardWindowTmux(ctx context.Context, client *tmuxctl.Client, session, windowName string, command []string) error {
	targetSession := strings.TrimSpace(session)
	if targetSession == "" {
		current, err := client.CurrentSession(ctx)
		if err != nil {
			return err
		}
		targetSession = current
	}
	if strings.TrimSpace(targetSession) == "" {
		return fmt.Errorf("no active tmux session")
	}
	windows, err := client.ListWindows(ctx, targetSession)
	if err != nil {
		return err
	}
	for _, w := range windows {
		if w.Name == windowName {
			return client.SelectWindow(ctx, fmt.Sprintf("%s:%s", targetSession, windowName))
		}
	}
	args := []string{"new-window", "-t", targetSession, "-n", windowName}
	args = append(args, command...)
	cmd := exec.CommandContext(ctx, client.Binary(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return client.SelectWindow(ctx, fmt.Sprintf("%s:%s", targetSession, windowName))
}
