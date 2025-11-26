package tmuxctl

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SessionSnapshot contains a lightweight description of windows/panes for a tmux session.
type SessionSnapshot struct {
	Session string
	Windows []WindowSnapshot
}

// WindowSnapshot describes a tmux window and its panes.
type WindowSnapshot struct {
	Index  string
	Name   string
	Active bool
	Panes  []PaneSnapshot
}

// PaneSnapshot describes a tmux pane.
type PaneSnapshot struct {
	Index  string
	Title  string
	Active bool
}

// SessionSnapshot fetches a snapshot of windows/panes for the given session.
func (c *Client) SessionSnapshot(ctx context.Context, session string) (SessionSnapshot, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return SessionSnapshot{}, fmt.Errorf("session is required")
	}
	cmd := c.run(ctx, c.bin, "list-windows", "-t", session, "-F", "#{window_index}\t#{window_name}\t#{window_active}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return SessionSnapshot{Session: session}, nil
		}
		return SessionSnapshot{}, wrapTmuxErr("list-windows", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	snap := SessionSnapshot{Session: session}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		win := WindowSnapshot{
			Index:  parts[0],
			Name:   parts[1],
			Active: parts[2] == "1",
		}
		panes, err := c.listPanes(ctx, fmt.Sprintf("%s:%s", session, win.Index))
		if err != nil {
			return SessionSnapshot{}, err
		}
		win.Panes = panes
		snap.Windows = append(snap.Windows, win)
	}
	return snap, nil
}

func (c *Client) listPanes(ctx context.Context, target string) ([]PaneSnapshot, error) {
	cmd := c.run(ctx, c.bin, "list-panes", "-t", target, "-F", "#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, wrapTmuxErr("list-panes", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var panes []PaneSnapshot
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		title := parts[2]
		if strings.TrimSpace(title) == "" {
			title = parts[3]
		}
		panes = append(panes, PaneSnapshot{
			Index:  parts[0],
			Active: parts[1] == "1",
			Title:  title,
		})
	}
	return panes, nil
}
