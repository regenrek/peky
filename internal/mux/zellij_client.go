package mux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/regenrek/peakypanes/internal/zellijctl"
)

// ZellijClient adapts zellijctl to the mux Client interface.
type ZellijClient struct {
	client *zellijctl.Client
}

func NewZellijClient(zellijPath, bridgePath string) (*ZellijClient, error) {
	client, err := zellijctl.NewClient(zellijPath, bridgePath)
	if err != nil {
		return nil, err
	}
	return &ZellijClient{client: client}, nil
}

func (z *ZellijClient) Type() Type {
	return Zellij
}

func (z *ZellijClient) Binary() string {
	return z.client.Binary()
}

func (z *ZellijClient) IsInside() bool {
	return insideZellij()
}

func (z *ZellijClient) ListSessions(ctx context.Context) ([]string, error) {
	return z.client.ListSessions(ctx)
}

func (z *ZellijClient) ListSessionsInfo(ctx context.Context) ([]SessionInfo, error) {
	sessions, err := z.client.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	paths, _ := zellijctl.LoadSessionPaths()
	out := make([]SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, SessionInfo{Name: s.Name, Path: paths[s.Name]})
	}
	return out, nil
}

func (z *ZellijClient) CurrentSession(ctx context.Context) (string, error) {
	if name := strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME")); name != "" {
		return name, nil
	}
	sessions, err := z.client.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range sessions {
		if s.IsCurrentSession {
			return s.Name, nil
		}
	}
	return "", nil
}

func (z *ZellijClient) ListWindows(ctx context.Context, session string) ([]WindowInfo, error) {
	sessions, err := z.client.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		if s.Name != session {
			continue
		}
		out := make([]WindowInfo, 0, len(s.Tabs))
		for _, tab := range s.Tabs {
			out = append(out, WindowInfo{
				Index:  strconv.Itoa(tab.Position),
				Name:   tab.Name,
				Active: tab.Active,
			})
		}
		return out, nil
	}
	return nil, nil
}

func (z *ZellijClient) ListPanesDetailed(ctx context.Context, target string) ([]PaneInfo, error) {
	session, window := splitTarget(target)
	sessions, err := z.client.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		if s.Name != session {
			continue
		}
		tabPos := -1
		if strings.TrimSpace(window) != "" {
			pos, err := strconv.Atoi(window)
			if err == nil {
				tabPos = pos
			}
		}
		if tabPos < 0 {
			for _, tab := range s.Tabs {
				if tab.Active {
					tabPos = tab.Position
					break
				}
			}
		}
		if tabPos < 0 {
			return nil, nil
		}
		panes := s.Panes.Panes[strconv.Itoa(tabPos)]
		out := make([]PaneInfo, 0, len(panes))
		for _, p := range panes {
			if p.IsPlugin || p.IsSuppressed || !p.IsSelectable {
				continue
			}
			command := ""
			if p.TerminalCommand != nil {
				command = *p.TerminalCommand
			}
			deadStatus := 0
			if p.ExitStatus != nil {
				deadStatus = *p.ExitStatus
			}
			out = append(out, PaneInfo{
				ID:         fmt.Sprintf("%s:%d", session, p.ID),
				Index:      strconv.Itoa(p.ID),
				Active:     p.IsFocused,
				Title:      p.Title,
				Command:    command,
				Left:       p.PaneContentX,
				Top:        p.PaneContentY,
				Width:      p.PaneContentColumns,
				Height:     p.PaneContentRows,
				Dead:       p.Exited,
				DeadStatus: deadStatus,
			})
		}
		return out, nil
	}
	return nil, nil
}

func (z *ZellijClient) CapturePaneLines(ctx context.Context, target string, lines int) ([]string, error) {
	session, paneID, err := splitSessionPane(target)
	if err != nil {
		return nil, err
	}
	return z.client.CapturePaneLines(ctx, session, paneID, lines)
}

func (z *ZellijClient) SessionHasClients(ctx context.Context, session string) (bool, error) {
	sessions, err := z.client.Snapshot(ctx)
	if err != nil {
		return false, err
	}
	for _, s := range sessions {
		if s.Name == session {
			return s.ConnectedClients > 0, nil
		}
	}
	return false, nil
}

func (z *ZellijClient) RenameSession(ctx context.Context, session, newName string) error {
	return z.client.RenameSession(ctx, session, newName)
}

func (z *ZellijClient) RenameWindow(ctx context.Context, session, windowTarget, newName string) error {
	pos, err := zellijctl.ParseTabPosition(windowTarget)
	if err != nil {
		return err
	}
	return z.client.RenameTab(ctx, session, pos, newName)
}

func (z *ZellijClient) KillSession(ctx context.Context, session string) error {
	return z.client.KillSession(ctx, session)
}

func (z *ZellijClient) SendKeys(ctx context.Context, target string, keys ...string) error {
	session, paneID, err := splitSessionPane(target)
	if err != nil {
		return err
	}
	text := keysToText(keys)
	if text == "" {
		return nil
	}
	return z.client.SendKeys(ctx, session, paneID, text)
}

func (z *ZellijClient) Attach(ctx context.Context, target string, inside bool) error {
	session, window := splitTarget(target)
	if strings.TrimSpace(session) == "" {
		return errors.New("session is required")
	}
	if inside {
		current := strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME"))
		if current == "" {
			current = session
		}
		var tabPos *int
		if strings.TrimSpace(window) != "" {
			if pos, err := strconv.Atoi(window); err == nil {
				tabPos = &pos
			}
		}
		return z.client.SwitchSession(ctx, current, session, tabPos)
	}
	return z.client.AttachSession(ctx, session)
}

func (z *ZellijClient) AttachCommand(target string, inside bool) (string, []string, []string) {
	session, _ := splitTarget(target)
	args := []string{"attach", session}
	return "zellij", args, nil
}

func (z *ZellijClient) SupportsPopup(ctx context.Context) bool {
	return true
}

func (z *ZellijClient) DisplayPopup(ctx context.Context, opts PopupOptions, command []string) error {
	args := []string{}
	if session := strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME")); session != "" {
		args = append(args, "--session", session)
	}
	args = append(args, "run", "--floating", "--close-on-exit")
	if strings.TrimSpace(opts.StartDir) != "" {
		args = append(args, "--cwd", opts.StartDir)
	}
	if strings.TrimSpace(opts.Width) != "" {
		args = append(args, "--width", opts.Width)
	}
	if strings.TrimSpace(opts.Height) != "" {
		args = append(args, "--height", opts.Height)
	}
	args = append(args, "--")
	args = append(args, command...)
	cmd := exec.CommandContext(ctx, z.client.Binary(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (z *ZellijClient) OpenDashboardWindow(ctx context.Context, session, windowName string, command []string) error {
	if strings.TrimSpace(session) == "" {
		current, err := z.CurrentSession(ctx)
		if err != nil {
			return err
		}
		session = current
	}
	if strings.TrimSpace(session) == "" {
		return errors.New("no active zellij session")
	}
	windows, err := z.ListWindows(ctx, session)
	if err != nil {
		return err
	}
	for _, w := range windows {
		if w.Name == windowName {
			return z.runAction(ctx, session, "go-to-tab-name", windowName)
		}
	}
	args := []string{"--session", session, "action", "new-tab", "--name", windowName, "--"}
	args = append(args, command...)
	cmd := exec.CommandContext(ctx, z.client.Binary(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (z *ZellijClient) runAction(ctx context.Context, session string, action string, args ...string) error {
	cmdArgs := []string{"--session", session, "action", action}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, z.client.Binary(), cmdArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("zellij action %s: %w (%s)", action, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func insideZellij() bool {
	return strings.TrimSpace(os.Getenv("ZELLIJ")) != "" || strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME")) != ""
}

func splitTarget(target string) (string, string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", ""
	}
	parts := strings.SplitN(target, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return target, ""
}

func splitSessionPane(target string) (string, int, error) {
	session, pane := splitTarget(target)
	if strings.TrimSpace(session) == "" {
		return "", 0, fmt.Errorf("session is required")
	}
	pane = strings.TrimSpace(pane)
	if pane == "" {
		return "", 0, fmt.Errorf("pane id is required")
	}
	paneID, err := strconv.Atoi(pane)
	if err != nil {
		return "", 0, fmt.Errorf("invalid pane id %q", pane)
	}
	return session, paneID, nil
}

func keysToText(keys []string) string {
	var sb strings.Builder
	for _, key := range keys {
		switch key {
		case "C-m", "Enter":
			sb.WriteString("\n")
		case "C-c":
			sb.WriteString("\x03")
		default:
			sb.WriteString(key)
		}
	}
	return sb.String()
}
