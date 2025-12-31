package clone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/dashboard"
	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
)

// Register registers clone handlers.
func Register(reg *root.Registry) {
	reg.Register("clone", runClone)
}

func runClone(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("clone", ctx.Deps.Version)
	repo := strings.TrimSpace(ctx.Cmd.StringArg("repo"))
	if repo == "" {
		return fmt.Errorf("repository is required")
	}
	url := normalizeRepoURL(repo)
	name := extractRepoName(url)
	if name == "" {
		name = "repo"
	}
	path := strings.TrimSpace(ctx.Cmd.String("path"))
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home dir: %w", err)
		}
		path = filepath.Join(home, "projects", name)
	}
	cleanPath, err := ensureClonePath(path)
	if err != nil {
		return err
	}

	if _, err := os.Stat(cleanPath); err != nil {
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctxTimeout, "git", "clone", url, cleanPath)
		cmd.Stdout = ctx.Out
		cmd.Stderr = ctx.ErrOut
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("clone failed: %w", err)
		}
	}
	if err := startSession(ctx, cleanPath); err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action:  "clone",
			Status:  "ok",
			Targets: []output.TargetRef{{Type: "project", ID: cleanPath}},
			Details: map[string]any{
				"path": cleanPath,
				"url":  url,
			},
		})
	}
	return nil
}

func startSession(ctx root.CommandContext, projectPath string) error {
	if ctx.JSON {
		connect := ctx.Deps.Connect
		if connect == nil {
			return fmt.Errorf("daemon connection not configured")
		}
		ctxTimeout, cancel := context.WithTimeout(ctx.Context, 10*time.Second)
		client, err := connect(ctxTimeout, ctx.Deps.Version)
		if err != nil {
			cancel()
			return err
		}
		defer func() {
			cancel()
			_ = client.Close()
		}()
		_, err = client.StartSession(ctxTimeout, sessiond.StartSessionRequest{
			Name:       strings.TrimSpace(ctx.Cmd.String("session")),
			Path:       projectPath,
			LayoutName: strings.TrimSpace(ctx.Cmd.String("layout")),
		})
		return err
	}
	autoStart := &app.AutoStartSpec{
		Session: strings.TrimSpace(ctx.Cmd.String("session")),
		Path:    projectPath,
		Layout:  strings.TrimSpace(ctx.Cmd.String("layout")),
		Focus:   true,
	}
	return dashboard.Run(ctx, autoStart)
}

func normalizeRepoURL(repo string) string {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return repo
	}
	if strings.Contains(repo, "://") || strings.HasPrefix(repo, "git@") {
		return repo
	}
	return "https://github.com/" + repo
}

func extractRepoName(url string) string {
	url = strings.TrimSpace(strings.TrimSuffix(url, ".git"))
	url = strings.TrimSuffix(url, "/")
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func ensureClonePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	return abs, nil
}
