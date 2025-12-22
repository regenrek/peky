package tmuxctl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kregenrek/tmuxman/internal/layout"
)

// Client coordinates tmux operations to create deterministic pane grids.
type Client struct {
	bin string
	run func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// Options configures how a session should be created.
type Options struct {
	Session  string
	Layout   layout.Grid
	StartDir string
	Attach   bool
	Timeout  time.Duration
}

// Result reports what happened while satisfying the request.
type Result struct {
	Created  bool
	Attached bool
}

// NewClient resolves the tmux binary and returns a Client.
func NewClient(tmuxPath string) (*Client, error) {
	if tmuxPath == "" {
		var err error
		tmuxPath, err = exec.LookPath("tmux")
		if err != nil {
			return nil, fmt.Errorf("tmux not found in PATH: %w", err)
		}
	}
	return &Client{bin: tmuxPath, run: exec.CommandContext}, nil
}

// Binary returns the resolved tmux binary path.
func (c *Client) Binary() string {
	return c.bin
}

// WithExec allows tests to override the exec implementation.
func (c *Client) WithExec(fn func(context.Context, string, ...string) *exec.Cmd) {
	c.run = fn
}

// EnsureSession creates the session if missing and optionally attaches.
func (c *Client) EnsureSession(ctx context.Context, opts Options) (Result, error) {
	if opts.Session == "" {
		return Result{}, errors.New("session name is required")
	}
	if opts.Layout == (layout.Grid{}) {
		opts.Layout = layout.Default
	}
	if err := opts.Layout.Validate(); err != nil {
		return Result{}, err
	}
	if opts.StartDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Result{}, fmt.Errorf("determine working directory: %w", err)
		}
		opts.StartDir = wd
	}
	startDir, err := filepath.Abs(opts.StartDir)
	if err != nil {
		return Result{}, fmt.Errorf("resolve start dir: %w", err)
	}
	if stat, err := os.Stat(startDir); err != nil || !stat.IsDir() {
		return Result{}, fmt.Errorf("start dir %q does not exist", startDir)
	}

	setupCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		setupCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	exists, err := c.sessionExists(setupCtx, opts.Session)
	if err != nil {
		return Result{}, err
	}

	res := Result{}
	if !exists {
		if err := c.createGrid(setupCtx, opts.Session, startDir, opts.Layout); err != nil {
			return Result{}, err
		}
		res.Created = true
	}

	if !opts.Attach {
		return res, nil
	}

	if err := c.attach(context.Background(), opts.Session); err != nil {
		return res, err
	}
	res.Attached = true
	return res, nil
}

// ListSessions returns the names of tmux sessions currently running. When no
// server is running, the returned slice is empty and the error is nil.
func (c *Client) ListSessions(ctx context.Context) ([]string, error) {
	cmd := c.run(ctx, c.bin, "list-sessions", "-F", "#{session_name}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Check tmux output and the error message for benign "no server" cases.
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
	var sessions []string
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			sessions = append(sessions, name)
		}
	}
	return sessions, nil
}

// SourceFile loads tmux commands from the provided file path.
func (c *Client) SourceFile(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("tmux config path cannot be empty")
	}
	cmd := c.run(ctx, c.bin, "source-file", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("source-file", err, out)
	}
	return nil
}

// AttachExisting switches/attaches to an existing session, returning an error
// if the session is missing.
func (c *Client) AttachExisting(ctx context.Context, session string) error {
	if session == "" {
		return errors.New("session name is required to resume")
	}
	exists, err := c.sessionExists(ctx, session)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("tmux session %q not found", session)
	}
	return c.attach(context.Background(), session)
}

// CurrentSession returns the name of the currently attached tmux session, if any.
// When no tmux server is running, an empty string is returned and the error is nil.
func (c *Client) CurrentSession(ctx context.Context) (string, error) {
	cmd := c.run(ctx, c.bin, "display-message", "-p", "#S")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if msg == "" {
			msg = strings.ToLower(strings.TrimSpace(err.Error()))
		}
		if strings.Contains(msg, "no server") ||
			strings.Contains(msg, "failed to connect") ||
			strings.Contains(msg, "error connecting to") ||
			strings.Contains(msg, "no current target") {
			return "", nil
		}
		return "", wrapTmuxErr("display-message", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// KillSession terminates a tmux session by name.
func (c *Client) KillSession(ctx context.Context, session string) error {
	if session == "" {
		return errors.New("session name is required")
	}
	cmd := c.run(ctx, c.bin, "kill-session", "-t", session)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("kill-session", err, out)
	}
	return nil
}

// NewWindow creates a new tmux window in the given session. If windowName is
// non-empty, the window will be renamed accordingly. If startDir is non-empty,
// tmux will start the window in that directory. command, when non-empty, is
// passed as a single tmux "shell-command" argument, so it may contain spaces.
func (c *Client) NewWindow(ctx context.Context, session, windowName, startDir, command string) error {
	if session == "" {
		return errors.New("session name is required")
	}
	args := []string{"new-window", "-t", session}
	if windowName != "" {
		args = append(args, "-n", windowName)
	}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if command != "" {
		args = append(args, command)
	}
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("new-window", err, out)
	}
	return nil
}

// KillWindow removes a window named windowName from the given session. If the
// window does not exist, the error is ignored so callers can treat the operation
// as idempotent.
func (c *Client) KillWindow(ctx context.Context, session, windowName string) error {
	if session == "" {
		return errors.New("session name is required")
	}
	if windowName == "" {
		return errors.New("window name is required")
	}
	target := fmt.Sprintf("%s:%s", session, windowName)
	cmd := c.run(ctx, c.bin, "kill-window", "-t", target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			msg := strings.ToLower(strings.TrimSpace(string(out)))
			if strings.Contains(msg, "can't find window") {
				return nil
			}
		}
		return wrapTmuxErr("kill-window", err, out)
	}
	return nil
}

// SplitWindow splits the target pane or window. If vertical is true, a vertical
// split is created (top/bottom panes); otherwise a horizontal split (left/right).
// When percent is greater than zero, it is passed to tmux via -p to control pane size.
func (c *Client) SplitWindow(ctx context.Context, target, startDir string, vertical bool, percent int) error {
	if target == "" {
		return errors.New("split target cannot be empty")
	}
	orientation := "-h"
	if vertical {
		orientation = "-v"
	}
	args := []string{"split-window", orientation, "-t", target}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if percent > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", percent))
	}
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("split-window", err, out)
	}
	return nil
}

func (c *Client) sessionExists(ctx context.Context, session string) (bool, error) {
	cmd := c.run(ctx, c.bin, "has-session", "-t", session)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, wrapTmuxErr("has-session", err, out)
	}
	return true, nil
}

func (c *Client) createGrid(ctx context.Context, session, startDir string, grid layout.Grid) error {
	firstPane, err := c.newSession(ctx, session, startDir)
	if err != nil {
		return err
	}

	// Set remain-on-exit as default - off lets panes close normally when commands exit
	_ = c.SetOption(ctx, session, "remain-on-exit", "off")

	rowRoots := []string{firstPane}
	current := firstPane
	for i := 1; i < grid.Rows; i++ {
		nextPane, err := c.splitVertical(ctx, current, startDir)
		if err != nil {
			return err
		}
		rowRoots = append(rowRoots, nextPane)
		current = nextPane
	}
	for _, rowPane := range rowRoots {
		target := rowPane
		for i := 1; i < grid.Columns; i++ {
			nextPane, err := c.splitHorizontal(ctx, target, startDir)
			if err != nil {
				return err
			}
			target = nextPane
		}
	}
	return c.equalize(ctx, session)
}

func (c *Client) newSession(ctx context.Context, session, startDir string) (string, error) {
	args := []string{"new-session", "-d", "-s", session, "-c", startDir, "-P", "-F", "#{pane_id}"}
	cmd := c.run(ctx, c.bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", wrapTmuxErr("new-session", err, nil)
	}
	pane := strings.TrimSpace(string(out))
	if pane == "" {
		return "", errors.New("tmux new-session returned empty pane id")
	}
	return pane, nil
}

func (c *Client) splitVertical(ctx context.Context, target, startDir string) (string, error) {
	return c.splitPane(ctx, target, startDir, "-v")
}

func (c *Client) splitHorizontal(ctx context.Context, target, startDir string) (string, error) {
	return c.splitPane(ctx, target, startDir, "-h")
}

func (c *Client) splitPane(ctx context.Context, target, startDir, orientation string) (string, error) {
	if target == "" {
		return "", errors.New("split target pane cannot be empty")
	}
	args := []string{"split-window", orientation, "-t", target, "-c", startDir, "-P", "-F", "#{pane_id}"}
	cmd := c.run(ctx, c.bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", wrapTmuxErr(fmt.Sprintf("split-window %s", orientation), err, nil)
	}
	pane := strings.TrimSpace(string(out))
	if pane == "" {
		return "", errors.New("tmux split-window returned empty pane id")
	}
	return pane, nil
}

func (c *Client) equalize(ctx context.Context, session string) error {
	target := session
	if strings.TrimSpace(target) == "" {
		return errors.New("session cannot be empty for select-layout")
	}
	cmd := c.run(ctx, c.bin, "select-layout", "-t", target, "tiled")
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("select-layout", err, out)
	}
	return nil
}

func (c *Client) attach(ctx context.Context, session string) error {
	if insideTmux() {
		return c.switchClient(ctx, session)
	}
	if err := c.attachSession(ctx, session); err != nil {
		if isNestedTmuxErr(err) {
			return c.switchClient(ctx, session)
		}
		return err
	}
	return nil
}

func (c *Client) attachSession(ctx context.Context, session string) error {
	cmd := c.run(ctx, c.bin, "attach-session", "-t", session)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return wrapTmuxErr("attach-session", err, nil)
	}
	return nil
}

func (c *Client) switchClient(ctx context.Context, session string) error {
	cmd := c.run(ctx, c.bin, "switch-client", "-t", session)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return wrapTmuxErr("switch-client", err, nil)
	}
	return nil
}

func insideTmux() bool {
	if os.Getenv("TMUX") != "" || os.Getenv("TMUX_PANE") != "" {
		return true
	}
	return false
}

func wrapTmuxErr(subcmd string, err error, combined []byte) error {
	msg := strings.TrimSpace(string(combined))
	if msg == "" {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			msg = strings.TrimSpace(string(exitErr.Stderr))
		}
	}
	if msg == "" {
		msg = err.Error()
	}
	return fmt.Errorf("tmux %s: %s", subcmd, msg)
}

func isNestedTmuxErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nested")
}

// SetOption sets a tmux option for a session. Use "-g" as session for global options.
func (c *Client) SetOption(ctx context.Context, session, option, value string) error {
	args := []string{"set-option"}
	if session != "" && session != "-g" {
		args = append(args, "-t", session)
	} else if session == "-g" {
		args = append(args, "-g")
	}
	args = append(args, option, value)
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("set-option", err, out)
	}
	return nil
}

// BindKey registers a tmux key binding using the prefix table.
func (c *Client) BindKey(ctx context.Context, key, action string) error {
	key = strings.TrimSpace(key)
	action = strings.TrimSpace(action)
	if key == "" || action == "" {
		return errors.New("bind-key requires key and action")
	}
	args := []string{"bind-key", key}
	args = append(args, strings.Fields(action)...)
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("bind-key", err, out)
	}
	return nil
}

// SendKeys sends keystrokes to a target pane.
func (c *Client) SendKeys(ctx context.Context, target string, keys ...string) error {
	if target == "" {
		return errors.New("send-keys target cannot be empty")
	}
	args := []string{"send-keys", "-t", target}
	args = append(args, keys...)
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("send-keys", err, out)
	}
	return nil
}

// SelectPane sets the title of a pane.
func (c *Client) SelectPane(ctx context.Context, target, title string) error {
	if target == "" {
		return errors.New("select-pane target cannot be empty")
	}
	args := []string{"select-pane", "-t", target, "-T", title}
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("select-pane", err, out)
	}
	return nil
}

// SelectLayout applies a tmux layout to a window.
func (c *Client) SelectLayout(ctx context.Context, target, layoutName string) error {
	if target == "" {
		return errors.New("select-layout target cannot be empty")
	}
	args := []string{"select-layout", "-t", target, layoutName}
	cmd := c.run(ctx, c.bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapTmuxErr("select-layout", err, out)
	}
	return nil
}

// SplitWindowWithCmd splits a pane and optionally runs a command.
func (c *Client) SplitWindowWithCmd(ctx context.Context, target, startDir string, vertical bool, percent int, command string) (string, error) {
	if target == "" {
		return "", errors.New("split target cannot be empty")
	}
	orientation := "-h"
	if vertical {
		orientation = "-v"
	}
	args := []string{"split-window", orientation, "-t", target, "-P", "-F", "#{pane_id}"}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if percent > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", percent))
	}
	if command != "" {
		args = append(args, command)
	}
	cmd := c.run(ctx, c.bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", wrapTmuxErr("split-window", err, nil)
	}
	return strings.TrimSpace(string(out)), nil
}

// NewSessionWithCmd creates a new session and returns the first pane ID.
func (c *Client) NewSessionWithCmd(ctx context.Context, session, startDir, windowName, command string) (string, error) {
	if session == "" {
		return "", errors.New("session name is required")
	}
	args := []string{"new-session", "-d", "-s", session, "-P", "-F", "#{pane_id}"}
	if windowName != "" {
		args = append(args, "-n", windowName)
	}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if command != "" {
		args = append(args, command)
	}
	cmd := c.run(ctx, c.bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", wrapTmuxErr("new-session", err, nil)
	}
	return strings.TrimSpace(string(out)), nil
}

// NewWindowWithCmd creates a window and returns the pane ID.
func (c *Client) NewWindowWithCmd(ctx context.Context, session, windowName, startDir, command string) (string, error) {
	if session == "" {
		return "", errors.New("session name is required")
	}
	args := []string{"new-window", "-t", session, "-P", "-F", "#{pane_id}"}
	if windowName != "" {
		args = append(args, "-n", windowName)
	}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if command != "" {
		args = append(args, command)
	}
	cmd := c.run(ctx, c.bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", wrapTmuxErr("new-window", err, nil)
	}
	return strings.TrimSpace(string(out)), nil
}
