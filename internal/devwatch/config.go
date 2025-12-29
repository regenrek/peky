package devwatch

import (
	"errors"
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultInterval        = 300 * time.Millisecond
	defaultDebounce        = 200 * time.Millisecond
	defaultShutdownTimeout = 2 * time.Second
)

var (
	defaultBuildCmd = []string{"go", "build", "-o", "./peakypanes", "./cmd/peakypanes"}
	defaultRunCmd   = []string{"./peakypanes"}
	defaultWatch    = []string{"cmd", "internal", "assets", "go.mod", "go.sum"}
	defaultExts     = []string{".go", ".mod", ".sum", ".yml", ".yaml", ".json", ".toml", ".md"}
)

// Config defines the dev watch configuration.
type Config struct {
	BuildCmd        []string
	RunCmd          []string
	WatchPaths      []string
	Extensions      []string
	Interval        time.Duration
	Debounce        time.Duration
	ShutdownTimeout time.Duration
	Logger          *log.Logger
}

func prepareConfig(cfg Config) (Config, error) {
	if len(cfg.BuildCmd) == 0 {
		cfg.BuildCmd = append([]string{}, defaultBuildCmd...)
	}
	if len(cfg.RunCmd) == 0 {
		cfg.RunCmd = append([]string{}, defaultRunCmd...)
	}
	if len(cfg.WatchPaths) == 0 {
		cfg.WatchPaths = append([]string{}, defaultWatch...)
	}
	cfg.Extensions = normalizeExts(cfg.Extensions)
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.Debounce < 0 {
		cfg.Debounce = defaultDebounce
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stdout, "devwatch: ", log.LstdFlags)
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if len(cfg.BuildCmd) == 0 || strings.TrimSpace(cfg.BuildCmd[0]) == "" {
		return errors.New("devwatch: build command required")
	}
	if len(cfg.RunCmd) == 0 || strings.TrimSpace(cfg.RunCmd[0]) == "" {
		return errors.New("devwatch: run command required")
	}
	if len(cfg.WatchPaths) == 0 {
		return errors.New("devwatch: at least one watch path required")
	}
	return nil
}

func normalizeExts(exts []string) []string {
	if len(exts) == 0 {
		exts = defaultExts
	}
	seen := make(map[string]struct{}, len(exts))
	out := make([]string, 0, len(exts))
	for _, ext := range exts {
		normalized := normalizeExt(ext)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		out = append(out, ".go")
	}
	return out
}

func normalizeExt(ext string) string {
	value := strings.TrimSpace(ext)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, ".") {
		value = "." + value
	}
	return strings.ToLower(value)
}
