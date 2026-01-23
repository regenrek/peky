package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Installer runs channel-specific update commands.
type Installer struct {
	execCommand func(context.Context, string, ...string) *exec.Cmd
}

// NewInstaller builds a default installer.
func NewInstaller() Installer {
	return Installer{execCommand: exec.CommandContext}
}

// Install updates PeakyPanes using the detected channel.
func (i Installer) Install(ctx context.Context, spec InstallSpec) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	execCommand := i.execCommand
	if execCommand == nil {
		execCommand = exec.CommandContext
	}
	switch spec.Channel {
	case ChannelHomebrew:
		return runCommand(ctx, execCommand, "", "brew", "upgrade", "peakypanes")
	case ChannelNPMGlobal:
		return runCommand(ctx, execCommand, "", "npm", "update", "-g", "peakypanes")
	case ChannelNPMLocal:
		root, err := cleanRoot(spec.NPMRoot)
		if err != nil {
			return err
		}
		return runCommand(ctx, execCommand, root, "npm", "update", "peakypanes")
	case ChannelGit:
		root, err := cleanRoot(spec.GitRoot)
		if err != nil {
			return err
		}
		if err := runCommand(ctx, execCommand, root, "git", "pull", "--ff-only"); err != nil {
			return err
		}
		return runCommand(ctx, execCommand, root, "go", "install", "./cmd/peakypanes", "./cmd/peky")
	default:
		return fmt.Errorf("unsupported update channel: %s", spec.Channel)
	}
}

func cleanRoot(root string) (string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return "", fmt.Errorf("update root is empty")
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("update root must be absolute")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("update root unavailable: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("update root is not a directory")
	}
	return cleaned, nil
}

func runCommand(ctx context.Context, execCommand func(context.Context, string, ...string) *exec.Cmd, dir string, name string, args ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if execCommand == nil {
		execCommand = exec.CommandContext
	}
	cmd := execCommand(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("update command failed: %w", err)
		}
		return fmt.Errorf("update command failed: %w: %s", err, trimmed)
	}
	return nil
}
