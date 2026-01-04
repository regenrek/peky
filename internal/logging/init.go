package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/appdirs"
	"gopkg.in/natefinch/lumberjack.v2"
)

type InitOptions struct {
	App     string
	Version string
	Mode    Mode
}

type initResult struct {
	Logger *slog.Logger
	Close  func() error
}

func Init(ctx context.Context, cfg Config, opts InitOptions) (func() error, error) {
	if opts.App == "" {
		opts.App = "peakypanes"
	}
	if opts.Mode == 0 {
		opts.Mode = ModeCLI
	}

	base := DefaultConfig(opts.Mode)
	cfg = mergeConfig(base, cfg)
	cfg = cfg.WithEnv()
	normalized, err := cfg.Normalize()
	if err != nil {
		return nil, err
	}

	res, err := buildLogger(ctx, normalized, opts)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(res.Logger)
	setIncludePayloads(normalized.IncludePayloads != nil && *normalized.IncludePayloads)
	return res.Close, nil
}

func mergeConfig(base, override Config) Config {
	out := base
	if override.Level != nil {
		out.Level = override.Level
	}
	if override.Format != nil {
		out.Format = override.Format
	}
	if override.Sink != nil {
		out.Sink = override.Sink
	}
	if override.File != nil {
		out.File = override.File
	}
	if override.AddSource != nil {
		out.AddSource = override.AddSource
	}
	if override.IncludePayloads != nil {
		out.IncludePayloads = override.IncludePayloads
	}
	if override.MaxSizeMB != nil {
		out.MaxSizeMB = override.MaxSizeMB
	}
	if override.MaxBackups != nil {
		out.MaxBackups = override.MaxBackups
	}
	if override.MaxAgeDays != nil {
		out.MaxAgeDays = override.MaxAgeDays
	}
	if override.Compress != nil {
		out.Compress = override.Compress
	}
	return out
}

func buildLogger(ctx context.Context, cfg Config, opts InitOptions) (initResult, error) {
	level := parseLevel(cfg.Level)
	sink := SinkStderr
	if cfg.Sink != nil {
		sink = Sink(*cfg.Sink)
	}
	format := FormatText
	if cfg.Format != nil {
		format = Format(*cfg.Format)
	}
	addSource := cfg.AddSource != nil && *cfg.AddSource

	writer, closeFn, err := resolveWriter(cfg, sink)
	if err != nil {
		return initResult{}, err
	}
	handlerOpts := &slog.HandlerOptions{Level: level, AddSource: addSource}
	var handler slog.Handler
	switch format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, handlerOpts)
	default:
		handler = slog.NewTextHandler(writer, handlerOpts)
	}

	logger := slog.New(handler).With(
		slog.String("app", opts.App),
		slog.String("version", opts.Version),
		slog.String("mode", opts.Mode.String()),
	)
	return initResult{
		Logger: logger,
		Close:  closeFn,
	}, nil
}

func parseLevel(value *string) slog.Leveler {
	if value == nil {
		return slog.LevelInfo
	}
	switch strings.ToLower(strings.TrimSpace(*value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func resolveWriter(cfg Config, sink Sink) (io.Writer, func() error, error) {
	switch sink {
	case SinkNone:
		return io.Discard, func() error { return nil }, nil
	case SinkStderr:
		return os.Stderr, func() error { return nil }, nil
	case SinkFile:
		path := ""
		isOverride := false
		if cfg.File != nil {
			path = strings.TrimSpace(*cfg.File)
			isOverride = path != ""
		}
		if path == "" {
			dir, err := appdirs.RuntimeDir()
			if err != nil {
				return nil, nil, err
			}
			path = filepath.Join(dir, "daemon.log")
		}
		if err := ensureLogDir(filepath.Dir(path), isOverride); err != nil {
			return nil, nil, err
		}
		maxSize := derefInt(cfg.MaxSizeMB, 20)
		maxBackups := derefInt(cfg.MaxBackups, 5)
		maxAge := derefInt(cfg.MaxAgeDays, 7)
		compress := derefBool(cfg.Compress, true)

		rot := &lumberjack.Logger{
			Filename:   path,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}
		return rot, func() error { return rot.Close() }, nil
	default:
		return nil, nil, fmt.Errorf("logging: unknown sink %q", sink)
	}
}

func derefInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func derefBool(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
