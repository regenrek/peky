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

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	exists, err := c.sessionExists(ctx, opts.Session)
	if err != nil {
		return Result{}, err
	}

	res := Result{}
	if !exists {
		if err := c.createGrid(ctx, opts.Session, startDir, opts.Layout); err != nil {
			return Result{}, err
		}
		res.Created = true
	}

	if !opts.Attach {
		return res, nil
	}

	if err := c.attach(ctx, opts.Session); err != nil {
		return res, err
	}
	res.Attached = true
	return res, nil
}

func (c *Client) sessionExists(ctx context.Context, session string) (bool, error) {
	cmd := c.run(ctx, c.bin, "has-session", "-t", session)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("tmux has-session: %w", err)
	}
	return true, nil
}

func (c *Client) createGrid(ctx context.Context, session, startDir string, grid layout.Grid) error {
	firstPane, err := c.newSession(ctx, session, startDir)
	if err != nil {
		return err
	}
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
		return "", fmt.Errorf("tmux new-session: %w", err)
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
		return "", fmt.Errorf("tmux split-window %s: %w", orientation, err)
	}
	pane := strings.TrimSpace(string(out))
	if pane == "" {
		return "", errors.New("tmux split-window returned empty pane id")
	}
	return pane, nil
}

func (c *Client) equalize(ctx context.Context, session string) error {
	window := fmt.Sprintf("%s:0", session)
	cmd := c.run(ctx, c.bin, "select-layout", "-t", window, "tiled")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux select-layout tiled: %w", err)
	}
	return nil
}

func (c *Client) attach(ctx context.Context, session string) error {
	var args []string
	if os.Getenv("TMUX") == "" {
		args = []string{"attach-session", "-t", session}
	} else {
		args = []string{"switch-client", "-t", session}
	}
	cmd := c.run(ctx, c.bin, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return nil
}
