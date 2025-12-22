package mux

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/regenrek/peakypanes/internal/tmuxctl"
)

// TmuxClient adapts tmuxctl to the mux Client interface.
type TmuxClient struct {
	client *tmuxctl.Client
}

func NewTmuxClient(tmuxPath string) (*TmuxClient, error) {
	client, err := tmuxctl.NewClient(tmuxPath)
	if err != nil {
		return nil, err
	}
	return &TmuxClient{client: client}, nil
}

func (t *TmuxClient) WithExec(fn func(context.Context, string, ...string) *exec.Cmd) {
	t.client.WithExec(fn)
}

func (t *TmuxClient) Type() Type {
	return Tmux
}

func (t *TmuxClient) Binary() string {
	return t.client.Binary()
}

func (t *TmuxClient) IsInside() bool {
	return insideTmux()
}

func (t *TmuxClient) ListSessions(ctx context.Context) ([]string, error) {
	return t.client.ListSessions(ctx)
}

func (t *TmuxClient) ListSessionsInfo(ctx context.Context) ([]SessionInfo, error) {
	info, err := t.client.ListSessionsInfo(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SessionInfo, 0, len(info))
	for _, s := range info {
		out = append(out, SessionInfo{Name: s.Name, Path: s.Path})
	}
	return out, nil
}

func (t *TmuxClient) CurrentSession(ctx context.Context) (string, error) {
	return t.client.CurrentSession(ctx)
}

func (t *TmuxClient) ListWindows(ctx context.Context, session string) ([]WindowInfo, error) {
	windows, err := t.client.ListWindows(ctx, session)
	if err != nil {
		return nil, err
	}
	out := make([]WindowInfo, 0, len(windows))
	for _, w := range windows {
		out = append(out, WindowInfo{Index: w.Index, Name: w.Name, Active: w.Active})
	}
	return out, nil
}

func (t *TmuxClient) ListPanesDetailed(ctx context.Context, target string) ([]PaneInfo, error) {
	panes, err := t.client.ListPanesDetailed(ctx, target)
	if err != nil {
		return nil, err
	}
	out := make([]PaneInfo, 0, len(panes))
	for _, p := range panes {
		out = append(out, PaneInfo{
			ID:         p.ID,
			Index:      p.Index,
			Active:     p.Active,
			Title:      p.Title,
			Command:    p.Command,
			Left:       p.Left,
			Top:        p.Top,
			Width:      p.Width,
			Height:     p.Height,
			Dead:       p.Dead,
			DeadStatus: p.DeadStatus,
			LastActive: p.LastActive,
		})
	}
	return out, nil
}

func (t *TmuxClient) CapturePaneLines(ctx context.Context, target string, lines int) ([]string, error) {
	return t.client.CapturePaneLines(ctx, target, lines)
}

func (t *TmuxClient) SessionHasClients(ctx context.Context, session string) (bool, error) {
	return t.client.SessionHasClients(ctx, session)
}

func (t *TmuxClient) RenameSession(ctx context.Context, session, newName string) error {
	return t.client.RenameSession(ctx, session, newName)
}

func (t *TmuxClient) RenameWindow(ctx context.Context, session, windowTarget, newName string) error {
	return t.client.RenameWindow(ctx, session, windowTarget, newName)
}

func (t *TmuxClient) KillSession(ctx context.Context, session string) error {
	return t.client.KillSession(ctx, session)
}

func (t *TmuxClient) SendKeys(ctx context.Context, target string, keys ...string) error {
	return t.client.SendKeys(ctx, target, keys...)
}

func (t *TmuxClient) Attach(ctx context.Context, target string, inside bool) error {
	if inside {
		return t.client.SwitchClient(ctx, target)
	}
	return t.client.AttachSession(ctx, target)
}

func (t *TmuxClient) AttachCommand(target string, inside bool) (string, []string, []string) {
	args := []string{"attach-session", "-t", target}
	if inside {
		if socket := tmuxSocketFromEnv(os.Getenv("TMUX")); socket != "" {
			args = append([]string{"-S", socket}, args...)
		}
	}
	return "tmux", args, nil
}

func (t *TmuxClient) SupportsPopup(ctx context.Context) bool {
	return t.client.SupportsPopup(ctx)
}

func (t *TmuxClient) DisplayPopup(ctx context.Context, opts PopupOptions, command []string) error {
	return t.client.DisplayPopup(ctx, tmuxctl.PopupOptions{Width: opts.Width, Height: opts.Height, StartDir: opts.StartDir}, command)
}

func (t *TmuxClient) OpenDashboardWindow(ctx context.Context, session, windowName string, command []string) error {
	return openDashboardWindowTmux(ctx, t.client, session, windowName, command)
}

// SwitchClient exposes tmuxctl's switch-client for attach logic.
func (t *TmuxClient) SwitchClient(ctx context.Context, session string) error {
	return t.client.SwitchClient(ctx, session)
}

// AttachSession exposes tmuxctl's attach-session for attach logic.
func (t *TmuxClient) AttachSession(ctx context.Context, session string) error {
	return t.client.AttachSession(ctx, session)
}

func tmuxSocketFromEnv(env string) string {
	env = strings.TrimSpace(env)
	if env == "" {
		return ""
	}
	parts := strings.SplitN(env, ",", 2)
	return strings.TrimSpace(parts[0])
}
