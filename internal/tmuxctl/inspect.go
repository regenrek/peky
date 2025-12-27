package tmuxctl

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SessionInfo describes a tmux session with optional path metadata.
type SessionInfo struct {
	Name string
	Path string
}

// WindowInfo describes a tmux window.
type WindowInfo struct {
	Index  string
	Name   string
	Active bool
}

// PaneInfo describes a tmux pane with geometry and status metadata.
type PaneInfo struct {
	ID           string
	Index        string
	Active       bool
	Title        string
	Command      string
	StartCommand string
	PID          int
	Left         int
	Top          int
	Width        int
	Height       int
	Dead         bool
	DeadStatus   int
	LastActive   time.Time
}

// SessionHasClients returns true if the session has any attached clients.
func (c *Client) SessionHasClients(ctx context.Context, session string) (bool, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return false, fmt.Errorf("session is required")
	}
	cmd := c.run(ctx, c.bin, "list-clients", "-t", session, "-F", "#{client_tty}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if msg == "" {
			msg = strings.ToLower(strings.TrimSpace(err.Error()))
		}
		if strings.Contains(msg, "no server") ||
			strings.Contains(msg, "no such file") ||
			strings.Contains(msg, "failed to connect") ||
			strings.Contains(msg, "error connecting to") ||
			strings.Contains(msg, "no sessions") ||
			strings.Contains(msg, "no session") ||
			strings.Contains(msg, "can't find") {
			return false, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, wrapTmuxErr("list-clients", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true, nil
		}
	}
	return false, nil
}

// HasClientOnTTY returns true if any tmux client is attached to the given tty.
func (c *Client) HasClientOnTTY(ctx context.Context, tty string) (bool, error) {
	tty = strings.TrimSpace(tty)
	if tty == "" {
		return false, nil
	}
	cmd := c.run(ctx, c.bin, "list-clients", "-F", "#{client_tty}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if msg == "" {
			msg = strings.ToLower(strings.TrimSpace(err.Error()))
		}
		if strings.Contains(msg, "no server") ||
			strings.Contains(msg, "no such file") ||
			strings.Contains(msg, "failed to connect") ||
			strings.Contains(msg, "error connecting to") ||
			strings.Contains(msg, "no sessions") ||
			strings.Contains(msg, "no session") ||
			strings.Contains(msg, "can't find") {
			return false, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, wrapTmuxErr("list-clients", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == tty {
			return true, nil
		}
	}
	return false, nil
}

// ListSessionsInfo returns session names with their paths when available.
func (c *Client) ListSessionsInfo(ctx context.Context) ([]SessionInfo, error) {
	cmd := c.run(ctx, c.bin, "list-sessions", "-F", "#{session_name}\t#{session_path}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if msg == "" {
			msg = strings.ToLower(strings.TrimSpace(err.Error()))
		}
		if strings.Contains(msg, "no server") ||
			strings.Contains(msg, "no such file") ||
			strings.Contains(msg, "failed to connect") ||
			strings.Contains(msg, "error connecting to") ||
			strings.Contains(msg, "no sessions") {
			return nil, nil
		}
		return nil, wrapTmuxErr("list-sessions", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var sessions []SessionInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		name := strings.TrimSpace(parts[0])
		path := ""
		if len(parts) > 1 {
			path = strings.TrimSpace(parts[1])
			if strings.HasPrefix(path, "#{") {
				path = ""
			}
		}
		if name != "" {
			sessions = append(sessions, SessionInfo{Name: name, Path: path})
		}
	}
	return sessions, nil
}

// ListWindows returns windows for the given session.
func (c *Client) ListWindows(ctx context.Context, session string) ([]WindowInfo, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return nil, fmt.Errorf("session is required")
	}
	cmd := c.run(ctx, c.bin, "list-windows", "-t", session, "-F", "#{window_index}\t#{window_name}\t#{window_active}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, wrapTmuxErr("list-windows", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var windows []WindowInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		windows = append(windows, WindowInfo{
			Index:  strings.TrimSpace(parts[0]),
			Name:   strings.TrimSpace(parts[1]),
			Active: strings.TrimSpace(parts[2]) == "1",
		})
	}
	return windows, nil
}

// ListPanesDetailed returns panes for a window or pane target with geometry.
func (c *Client) ListPanesDetailed(ctx context.Context, target string) ([]PaneInfo, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("pane target is required")
	}
	fullFormat := strings.Join([]string{
		"#{pane_id}",
		"#{pane_index}",
		"#{pane_active}",
		"#{pane_title}",
		"#{pane_current_command}",
		"#{pane_start_command}",
		"#{pane_pid}",
		"#{pane_left}",
		"#{pane_top}",
		"#{pane_width}",
		"#{pane_height}",
		"#{pane_dead}",
		"#{pane_dead_status}",
		"#{pane_last_active}",
	}, "\t")

	out, err := c.run(ctx, c.bin, "list-panes", "-t", target, "-F", fullFormat).CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Try a legacy format without pane_pid before falling back.
			legacyFormat := strings.Join([]string{
				"#{pane_id}",
				"#{pane_index}",
				"#{pane_active}",
				"#{pane_title}",
				"#{pane_current_command}",
				"#{pane_start_command}",
				"#{pane_left}",
				"#{pane_top}",
				"#{pane_width}",
				"#{pane_height}",
				"#{pane_dead}",
				"#{pane_dead_status}",
				"#{pane_last_active}",
			}, "\t")
			out, err = c.run(ctx, c.bin, "list-panes", "-t", target, "-F", legacyFormat).CombinedOutput()
			if err != nil {
				return c.listPanesBasic(ctx, target)
			}
			panes := parsePanesFull(string(out))
			if len(panes) == 0 {
				return c.listPanesBasic(ctx, target)
			}
			return panes, nil
		}
		return nil, wrapTmuxErr("list-panes", err, out)
	}
	panes := parsePanesFull(string(out))
	if len(panes) == 0 {
		// Fallback to basic parse if full format didn't yield data.
		return c.listPanesBasic(ctx, target)
	}
	return panes, nil
}

func (c *Client) listPanesBasic(ctx context.Context, target string) ([]PaneInfo, error) {
	format := strings.Join([]string{
		"#{pane_id}",
		"#{pane_index}",
		"#{pane_active}",
		"#{pane_title}",
		"#{pane_current_command}",
		"#{pane_left}",
		"#{pane_top}",
		"#{pane_width}",
		"#{pane_height}",
	}, "\t")
	out, err := c.run(ctx, c.bin, "list-panes", "-t", target, "-F", format).CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, wrapTmuxErr("list-panes", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var panes []PaneInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 9 {
			continue
		}
		left := parseInt(parts[5])
		top := parseInt(parts[6])
		width := parseInt(parts[7])
		height := parseInt(parts[8])
		title := strings.TrimSpace(parts[3])
		command := strings.TrimSpace(parts[4])
		if title == "" {
			title = command
		}
		panes = append(panes, PaneInfo{
			ID:      strings.TrimSpace(parts[0]),
			Index:   strings.TrimSpace(parts[1]),
			Active:  strings.TrimSpace(parts[2]) == "1",
			Title:   title,
			Command: command,
			Left:    left,
			Top:     top,
			Width:   width,
			Height:  height,
		})
	}
	return panes, nil
}

func parsePanesFull(out string) []PaneInfo {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var panes []PaneInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 13 {
			continue
		}
		startCommand := strings.TrimSpace(parts[5])
		pid := 0
		leftIdx := 6
		deadIdx := 10
		lastActiveIdx := 12
		if len(parts) >= 14 {
			pid = parseInt(parts[6])
			leftIdx = 7
			deadIdx = 11
			lastActiveIdx = 13
		}
		left := parseInt(parts[leftIdx])
		top := parseInt(parts[leftIdx+1])
		width := parseInt(parts[leftIdx+2])
		height := parseInt(parts[leftIdx+3])
		deadStatus := parseInt(parts[deadIdx+1])
		lastActive := parseInt64(parts[lastActiveIdx])
		var lastActiveTime time.Time
		if lastActive > 0 {
			lastActiveTime = time.Unix(lastActive, 0)
		}
		title := strings.TrimSpace(parts[3])
		command := strings.TrimSpace(parts[4])
		if title == "" {
			title = command
		}
		panes = append(panes, PaneInfo{
			ID:           strings.TrimSpace(parts[0]),
			Index:        strings.TrimSpace(parts[1]),
			Active:       strings.TrimSpace(parts[2]) == "1",
			Title:        title,
			Command:      command,
			StartCommand: startCommand,
			PID:          pid,
			Left:         left,
			Top:          top,
			Width:        width,
			Height:       height,
			Dead:         strings.TrimSpace(parts[deadIdx]) == "1",
			DeadStatus:   deadStatus,
			LastActive:   lastActiveTime,
		})
	}
	return panes
}

// CapturePaneLines returns the last N lines of a pane buffer.
func (c *Client) CapturePaneLines(ctx context.Context, target string, lines int) ([]string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("pane target is required")
	}
	if lines <= 0 {
		return nil, nil
	}
	start := fmt.Sprintf("-%d", lines)
	parse := func(out []byte) []string {
		raw := strings.TrimRight(string(out), "\n")
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		return strings.Split(raw, "\n")
	}

	// Try alternate screen first (useful for full-screen TUIs).
	cmd := c.run(ctx, c.bin, "capture-pane", "-p", "-J", "-e", "-a", "-t", target, "-S", "0", "-E", "-")
	out, err := cmd.CombinedOutput()
	if err == nil {
		if parsed := parse(out); len(parsed) > 0 {
			if lines > 0 && len(parsed) > lines {
				return parsed[len(parsed)-lines:], nil
			}
			return parsed, nil
		}
	}

	// Fallback to default screen if -a is unsupported or returns empty content.
	cmd = c.run(ctx, c.bin, "capture-pane", "-p", "-J", "-e", "-t", target, "-S", start, "-E", "-")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return nil, wrapTmuxErr("capture-pane", err, out)
	}
	return parse(out), nil
}

func parseInt(v string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}

func parseInt64(v string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return n
}
