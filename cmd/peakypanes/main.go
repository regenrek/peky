package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/app"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/logging"
)

var version = "dev"

func main() {
	mode := logging.ModeFromArgs(os.Args)
	logCfg := logging.Config{}
	if configPath, err := layout.DefaultConfigPath(); err == nil && configPath != "" {
		if cfg, err := layout.LoadConfig(configPath); err == nil && cfg != nil {
			logCfg = cfg.Logging
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "peakypanes: load config: %v\n", err)
			os.Exit(1)
		}
	}
	closeLogger, err := logging.Init(context.Background(), logCfg, logging.InitOptions{
		App:     "peakypanes",
		Version: version,
		Mode:    mode,
	})
	if err != nil {
		if mode == logging.ModeDaemon {
			fmt.Fprintf(os.Stderr, "peakypanes: init logging: %v\n", err)
			os.Exit(1)
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
		slog.Error("init logging failed; using stderr fallback", "err", err)
	} else if closeLogger != nil {
		defer func() { _ = closeLogger() }()
	}

	deps := root.DefaultDependencies(version)
	runner, err := app.NewRunner(deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "peakypanes: %v\n", err)
		os.Exit(1)
	}
	if err := runner.Run(context.Background(), os.Args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "peakypanes: %v\n", err)
		os.Exit(1)
	}
}
