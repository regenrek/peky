package pekyconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	defaultProvider = "google"
	defaultModel    = "gemini-3-flash"
	defaultMaxDepth = 4
	defaultMaxItems = 500
)

var defaultBlockedCommands = []string{"daemon", "daemon.*"}

// Config represents ~/.peakypanes/config.toml.
type Config struct {
	Agent      AgentConfig      `toml:"agent"`
	QuickReply QuickReplyConfig `toml:"quick_reply"`
}

// AgentConfig configures the peky agent.
type AgentConfig struct {
	Provider        string   `toml:"provider"`
	Model           string   `toml:"model"`
	BlockedCommands []string `toml:"blocked_commands"`
	AllowedCommands []string `toml:"allowed_commands"`
}

// QuickReplyConfig configures quick reply behavior.
type QuickReplyConfig struct {
	Files QuickReplyFilesConfig `toml:"files"`
}

// QuickReplyFilesConfig configures @ file listing.
type QuickReplyFilesConfig struct {
	ShowHidden *bool `toml:"show_hidden"`
	MaxDepth   int   `toml:"max_depth"`
	MaxItems   int   `toml:"max_items"`
}

// Defaults returns the default configuration.
func Defaults() Config {
	return Config{
		Agent: AgentConfig{
			Provider:        defaultProvider,
			Model:           defaultModel,
			BlockedCommands: append([]string(nil), defaultBlockedCommands...),
		},
		QuickReply: QuickReplyConfig{
			Files: QuickReplyFilesConfig{
				ShowHidden: nil,
				MaxDepth:   defaultMaxDepth,
				MaxItems:   defaultMaxItems,
			},
		},
	}
}

// DefaultPath returns the default global config path (~/.peakypanes/config.toml).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".peakypanes", "config.toml"), nil
}

// Loader caches config values and reloads when the file changes.
type Loader struct {
	path     string
	lastRead fileState
	cached   Config
}

type fileState struct {
	modTime time.Time
	size    int64
}

// NewLoader creates a config loader for the provided path.
func NewLoader(path string) *Loader {
	return &Loader{
		path:   strings.TrimSpace(path),
		cached: Defaults(),
	}
}

// Load returns the cached config, reloading if the file changed.
func (l *Loader) Load() (Config, error) {
	if l == nil {
		return Defaults(), errors.New("nil loader")
	}
	path := strings.TrimSpace(l.path)
	if path == "" {
		return Defaults(), errors.New("empty config path")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			l.cached = Defaults()
			l.lastRead = fileState{}
			return l.cached, nil
		}
		return Defaults(), err
	}
	state := fileState{modTime: info.ModTime(), size: info.Size()}
	if state == l.lastRead {
		return l.cached, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Defaults(), err
	}
	cfg := Defaults()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Defaults(), err
	}
	applyDefaults(&cfg)
	l.cached = cfg
	l.lastRead = state
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.Agent.Provider) == "" {
		cfg.Agent.Provider = defaultProvider
	}
	if strings.TrimSpace(cfg.Agent.Model) == "" {
		cfg.Agent.Model = defaultModel
	}
	if cfg.QuickReply.Files.MaxDepth <= 0 {
		cfg.QuickReply.Files.MaxDepth = defaultMaxDepth
	}
	if cfg.QuickReply.Files.MaxItems <= 0 {
		cfg.QuickReply.Files.MaxItems = defaultMaxItems
	}
}
