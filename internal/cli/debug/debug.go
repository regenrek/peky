package debug

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/regenrek/peakypanes/internal/appdirs"
	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/runenv"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

// Register registers debug handlers.
func Register(reg *root.Registry) {
	reg.Register("debug.paths", runPaths)
}

func runPaths(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("debug.paths", ctx.Deps.Version)
	paths, err := resolvePaths()
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, paths)
	}
	if _, err := fmt.Fprintf(ctx.Out, "fresh_config: %v\n", paths.FreshConfig); err != nil {
		return err
	}
	lines := [8][2]string{
		{"runtime_dir", paths.RuntimeDir},
		{"data_dir", paths.DataDir},
		{"config_dir", paths.ConfigDir},
		{"config_path", paths.ConfigPath},
		{"layouts_dir", paths.LayoutsDir},
		{"daemon_socket_path", paths.DaemonSocketPath},
		{"daemon_pid_path", paths.DaemonPidPath},
		{"restart_notice_path", paths.RestartNoticePath},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(ctx.Out, "%s: %s\n", line[0], line[1]); err != nil {
			return err
		}
	}
	return nil
}

func resolvePaths() (output.DebugPaths, error) {
	runtimeDir, err := appdirs.RuntimeDirPath()
	if err != nil {
		return output.DebugPaths{}, err
	}
	dataDir, err := appdirs.DataDirPath()
	if err != nil {
		return output.DebugPaths{}, err
	}
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return output.DebugPaths{}, err
	}
	configDir := ""
	if configPath != "" {
		configDir = filepath.Dir(configPath)
	}
	layoutsDir, err := layout.DefaultLayoutsDir()
	if err != nil {
		return output.DebugPaths{}, err
	}
	socketPath := sessiond.ResolveSocketPath(runtimeDir)
	pidPath := sessiond.ResolvePidPath(runtimeDir)
	restartNoticePath := ""
	if configDir != "" {
		restartNoticePath = filepath.Join(configDir, identity.RestartNoticeFlagFile)
	}
	return output.DebugPaths{
		RuntimeDir:        runtimeDir,
		DataDir:           dataDir,
		ConfigDir:         configDir,
		ConfigPath:        configPath,
		LayoutsDir:        layoutsDir,
		DaemonSocketPath:  socketPath,
		DaemonPidPath:     pidPath,
		RestartNoticePath: restartNoticePath,
		FreshConfig:       runenv.FreshConfigEnabled(),
	}, nil
}
