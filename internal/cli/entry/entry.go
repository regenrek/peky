package entry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/app"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/logging"
)

// Run starts the CLI and returns the process exit code.
func Run(args []string, version string) int {
	appName := identity.CLIName
	mode := logging.ModeFromArgs(args)
	logCfg := logging.Config{}
	if configPath, err := layout.DefaultConfigPath(); err == nil && configPath != "" {
		if err := layout.EnsureDefaultGlobalConfig(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "%s: init config: %v\n", appName, err)
			return 1
		}
		if cfg, err := layout.LoadConfig(configPath); err == nil && cfg != nil {
			logCfg = cfg.Logging
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "%s: load config: %v\n", appName, err)
			return 1
		}
	}
	closeLogger, err := logging.Init(context.Background(), logCfg, logging.InitOptions{
		App:     identity.AppSlug,
		Version: version,
		Mode:    mode,
	})
	if err != nil {
		if mode == logging.ModeDaemon {
			fmt.Fprintf(os.Stderr, "%s: init logging: %v\n", appName, err)
			return 1
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
		slog.Error("init logging failed; using stderr fallback", "err", err)
	} else if closeLogger != nil {
		defer func() { _ = closeLogger() }()
	}

	deps := root.DefaultDependencies(version)
	deps.AppName = appName
	runner, err := app.NewRunner(deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		return 1
	}
	if err := runner.Run(context.Background(), args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		return 1
	}
	return 0
}
