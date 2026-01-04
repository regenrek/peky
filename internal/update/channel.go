package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Channel identifies the install/update channel.
type Channel string

const (
	ChannelUnknown   Channel = "unknown"
	ChannelNPMGlobal Channel = "npm_global"
	ChannelNPMLocal  Channel = "npm_local"
	ChannelHomebrew  Channel = "homebrew"
	ChannelGit       Channel = "git"
)

// InstallSpec captures channel-specific install metadata.
type InstallSpec struct {
	Channel    Channel
	Executable string
	NPMRoot    string
	GitRoot    string
}

// DetectInstall inspects the executable path to determine the install channel.
func DetectInstall(ctx context.Context, exePath string) (InstallSpec, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return InstallSpec{}, err
	}
	path := strings.TrimSpace(exePath)
	if path == "" {
		return InstallSpec{Channel: ChannelUnknown}, fmt.Errorf("executable path empty")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	resolved = filepath.Clean(resolved)
	slash := filepath.ToSlash(resolved)
	if isHomebrewPath(slash) {
		return InstallSpec{Channel: ChannelHomebrew, Executable: resolved}, nil
	}
	npmChannel, npmRoot := detectNPMChannel(slash)
	if npmChannel != ChannelUnknown {
		return InstallSpec{Channel: npmChannel, Executable: resolved, NPMRoot: npmRoot}, nil
	}
	gitRoot, ok := findGitRoot(ctx, filepath.Dir(resolved))
	if ok {
		return InstallSpec{Channel: ChannelGit, Executable: resolved, GitRoot: gitRoot}, nil
	}
	return InstallSpec{Channel: ChannelUnknown, Executable: resolved}, nil
}

func isHomebrewPath(path string) bool {
	return strings.Contains(path, "/Cellar/peakypanes/") || strings.Contains(path, "/Homebrew/Cellar/peakypanes/")
}

func detectNPMChannel(path string) (Channel, string) {
	idx := strings.LastIndex(path, "/node_modules/")
	if idx == -1 {
		return ChannelUnknown, ""
	}
	root := filepath.FromSlash(path[:idx])
	if strings.Contains(path, "/lib/node_modules/") {
		return ChannelNPMGlobal, root
	}
	return ChannelNPMLocal, root
}

func findGitRoot(ctx context.Context, start string) (string, bool) {
	path := filepath.Clean(start)
	for {
		if err := ctx.Err(); err != nil {
			return "", false
		}
		gitPath := filepath.Join(path, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() || info.Mode().IsRegular() {
				return path, true
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return "", false
}

// UpdateCommand returns a human-readable update command for the channel.
func UpdateCommand(spec InstallSpec) string {
	switch spec.Channel {
	case ChannelHomebrew:
		return "brew upgrade peakypanes"
	case ChannelNPMGlobal:
		return "npm update -g peakypanes"
	case ChannelNPMLocal:
		return "npm update peakypanes"
	case ChannelGit:
		return "git pull --ff-only && go install ./cmd/peakypanes ./cmd/peky"
	default:
		return "Update manually"
	}
}
