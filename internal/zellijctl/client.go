package zellijctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const pipeName = "peakypanes"

type Client struct {
	bin        string
	run        func(context.Context, string, ...string) *exec.Cmd
	bridgePath string

	cacheMu   sync.Mutex
	cacheAt   time.Time
	cacheData []sessionInfo
}

type pipeResponse struct {
	OK       bool          `json:"ok"`
	Error    string        `json:"error,omitempty"`
	Sessions []sessionInfo `json:"sessions,omitempty"`
	Lines    []string      `json:"lines,omitempty"`
}

type sessionInfo struct {
	Name             string         `json:"name"`
	Tabs             []tabInfo      `json:"tabs"`
	Panes            paneManifest   `json:"panes"`
	ConnectedClients int            `json:"connected_clients"`
	IsCurrentSession bool           `json:"is_current_session"`
	AvailableLayouts []interface{}  `json:"available_layouts,omitempty"`
	Plugins          map[string]any `json:"plugins,omitempty"`
}

type tabInfo struct {
	Position                     int    `json:"position"`
	Name                         string `json:"name"`
	Active                       bool   `json:"active"`
	SelectableTiledPanesCount    int    `json:"selectable_tiled_panes_count"`
	SelectableFloatingPanesCount int    `json:"selectable_floating_panes_count"`
}

type paneManifest struct {
	Panes map[string][]paneInfo `json:"panes"`
}

type paneInfo struct {
	ID                 int     `json:"id"`
	IsPlugin           bool    `json:"is_plugin"`
	IsFocused          bool    `json:"is_focused"`
	IsFloating         bool    `json:"is_floating"`
	IsSuppressed       bool    `json:"is_suppressed"`
	Title              string  `json:"title"`
	Exited             bool    `json:"exited"`
	ExitStatus         *int    `json:"exit_status"`
	PaneContentX       int     `json:"pane_content_x"`
	PaneContentY       int     `json:"pane_content_y"`
	PaneContentColumns int     `json:"pane_content_columns"`
	PaneContentRows    int     `json:"pane_content_rows"`
	TerminalCommand    *string `json:"terminal_command"`
	IsSelectable       bool    `json:"is_selectable"`
}

func NewClient(zellijPath, bridgePath string) (*Client, error) {
	if zellijPath == "" {
		var err error
		zellijPath, err = exec.LookPath("zellij")
		if err != nil {
			return nil, fmt.Errorf("zellij not found in PATH: %w", err)
		}
	}
	if strings.TrimSpace(bridgePath) == "" {
		path, err := EnsureBridgePlugin()
		if err != nil {
			return nil, err
		}
		bridgePath = path
	}
	return &Client{
		bin:        zellijPath,
		run:        exec.CommandContext,
		bridgePath: bridgePath,
	}, nil
}

func (c *Client) Binary() string {
	return c.bin
}

func (c *Client) BridgePath() string {
	return c.bridgePath
}

func (c *Client) WithExec(fn func(context.Context, string, ...string) *exec.Cmd) {
	c.run = fn
}

func (c *Client) ListSessions(ctx context.Context) ([]string, error) {
	cmd := c.run(ctx, c.bin, "list-sessions", "--short", "--no-formatting")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if msg == "" {
			msg = strings.ToLower(strings.TrimSpace(err.Error()))
		}
		if (strings.Contains(msg, "no active") && strings.Contains(msg, "sessions")) || strings.Contains(msg, "no sessions") {
			return nil, nil
		}
		return nil, fmt.Errorf("zellij list-sessions: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var sessions []string
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			sessions = append(sessions, name)
		}
	}
	return sessions, nil
}

func (c *Client) Snapshot(ctx context.Context) ([]sessionInfo, error) {
	c.cacheMu.Lock()
	if time.Since(c.cacheAt) < 250*time.Millisecond && c.cacheData != nil {
		cached := c.cacheData
		c.cacheMu.Unlock()
		return cached, nil
	}
	c.cacheMu.Unlock()

	sessions, err := c.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		c.cacheMu.Lock()
		c.cacheData = nil
		c.cacheAt = time.Now()
		c.cacheMu.Unlock()
		return nil, nil
	}

	var lastErr error
	for _, session := range sessions {
		resp, err := c.pipe(ctx, session, map[string]any{"action": "snapshot"})
		if err != nil {
			lastErr = err
			continue
		}
		if !resp.OK {
			lastErr = errors.New(resp.Error)
			continue
		}
		c.cacheMu.Lock()
		c.cacheData = resp.Sessions
		c.cacheAt = time.Now()
		cached := c.cacheData
		c.cacheMu.Unlock()
		return cached, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no zellij sessions responded to snapshot request")
	}
	return nil, lastErr
}

func (c *Client) CapturePaneLines(ctx context.Context, session string, paneID int, lines int) ([]string, error) {
	payload := map[string]any{
		"action":  "pane_scrollback",
		"pane_id": paneID,
		"lines":   lines,
	}
	resp, err := c.pipe(ctx, session, payload)
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	return resp.Lines, nil
}

func (c *Client) SendKeys(ctx context.Context, session string, paneID int, text string) error {
	payload := map[string]any{
		"action":  "send_keys",
		"pane_id": paneID,
		"text":    text,
	}
	resp, err := c.pipe(ctx, session, payload)
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	return nil
}

func (c *Client) RenameSession(ctx context.Context, session, newName string) error {
	payload := map[string]any{
		"action":   "rename_session",
		"new_name": newName,
	}
	resp, err := c.pipe(ctx, session, payload)
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	return nil
}

func (c *Client) RenameTab(ctx context.Context, session string, tabPosition int, newName string) error {
	payload := map[string]any{
		"action":       "rename_tab",
		"tab_position": tabPosition,
		"new_name":     newName,
	}
	resp, err := c.pipe(ctx, session, payload)
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	return nil
}

func (c *Client) SwitchSession(ctx context.Context, currentSession, targetSession string, tabPosition *int) error {
	payload := map[string]any{
		"action":  "switch_session",
		"session": targetSession,
	}
	if tabPosition != nil {
		payload["tab_position"] = *tabPosition
	}
	resp, err := c.pipe(ctx, currentSession, payload)
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	return nil
}

func (c *Client) KillSession(ctx context.Context, session string) error {
	cmd := c.run(ctx, c.bin, "kill-session", session)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("zellij kill-session: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c *Client) AttachSession(ctx context.Context, session string) error {
	cmd := c.run(ctx, c.bin, "attach", session)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zellij attach: %w", err)
	}
	return nil
}

func (c *Client) pipe(ctx context.Context, session string, payload map[string]any) (pipeResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return pipeResponse{}, err
	}
	args := []string{}
	if strings.TrimSpace(session) != "" {
		args = append(args, "--session", session)
	}
	args = append(args, "action", "pipe", "--name", pipeName)
	if strings.TrimSpace(c.bridgePath) != "" {
		args = append(args, "--plugin", normalizePluginURL(c.bridgePath))
	}
	args = append(args, "--", string(data))
	cmd := c.run(ctx, c.bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return pipeResponse{}, fmt.Errorf("zellij pipe: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return pipeResponse{}, fmt.Errorf("zellij pipe returned empty response")
	}
	var resp pipeResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return pipeResponse{}, fmt.Errorf("parse pipe response: %w", err)
	}
	return resp, nil
}

func ParseTabPosition(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("tab index is required")
	}
	return strconv.Atoi(raw)
}
