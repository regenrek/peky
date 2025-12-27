package tmuxctl

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// PanePipe returns the current pipe-pane command and pid for a pane.
// If no pipe is set, command is "" and pid is 0.
func (c *Client) PanePipe(ctx context.Context, paneID string) (string, int, error) {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return "", 0, fmt.Errorf("pane id is required")
	}
	cmd := c.run(ctx, c.bin, "display-message", "-p", "-t", paneID, "#{pane_pipe}\t#{pane_pipe_pid}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, wrapTmuxErr("display-message", err, out)
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", 0, nil
	}
	parts := strings.SplitN(line, "\t", 2)
	pipeCmd := strings.TrimSpace(parts[0])
	if strings.HasPrefix(pipeCmd, "#{") {
		pipeCmd = ""
	}
	pid := 0
	if len(parts) == 2 {
		pid, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	return pipeCmd, pid, nil
}

// PipePane sets or clears pipe-pane for a given pane.
// If shellCommand is empty, pipe-pane is disabled.
func (c *Client) PipePane(ctx context.Context, paneID string, shellCommand string) error {
	paneID = strings.TrimSpace(paneID)
	if paneID == "" {
		return fmt.Errorf("pane id is required")
	}
	args := []string{"pipe-pane", "-t", paneID}
	if strings.TrimSpace(shellCommand) != "" {
		args = append(args, shellCommand)
	}
	out, err := c.run(ctx, c.bin, args...).CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			msg := strings.ToLower(strings.TrimSpace(string(out)))
			if strings.Contains(msg, "can't find pane") || strings.Contains(msg, "no such pane") {
				return nil
			}
		}
		return wrapTmuxErr("pipe-pane", err, out)
	}
	return nil
}
