package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/identity"
)

const defaultGlobalConfigContent = `# peky - Global Configuration
# https://github.com/regenrek/peakypanes

zellij:
  # config: ~/.config/zellij/config.kdl
  # layout_dir: ~/.config/zellij/layouts
  # bridge_plugin: ~/.config/peky/zellij/peky-bridge.wasm

ghostty:
  config: ~/.config/ghostty/config

# Dashboard UI settings
# dashboard:
#   refresh_ms: 2000
#   preview_lines: 12
#   preview_compact: true  # remove blank lines for denser previews
#   thumbnail_lines: 1
#   idle_seconds: 20
#   show_thumbnails: true
#   resize:
#     mouse_apply: live  # live | commit
#     mouse_throttle_ms: 16
#     freeze_content_during_drag: true
#   attach_behavior: current  # current | detached
#   project_roots:
#     - ~/projects
#   project_roots_allow_nongit: true
#   status_regex:
#     success: "(?i)done|finished|success|completed|✅"
#     error: "(?i)error|failed|panic|❌"
#     running: "(?i)running|in progress|building|installing|▶"
#   agent_detection:
#     codex: true
#     claude: true

# peky agent settings (used by /peky and Shift+Tab)
# agent:
#   provider: google
#   model: gemini-3-flash
#   # If allowed_commands is set, only these commands may run.
#   # allowed_commands:
#   #   - pane.add
#   #   - pane.split
#   #   - session.close
#   # Otherwise, blocked_commands denies specific commands/prefixes.
#   blocked_commands:
#     - daemon
#     - daemon.*

# Action line settings (quick reply; for @file picker)
# quick_reply:
#   enabled: false
#   files:
#     show_hidden: false
#     max_depth: 4
#     max_items: 500

# Load additional layouts from this directory
layout_dirs:
  - ~/.config/peky/layouts

# Define projects for quick access
# projects:
#   - name: my-project
#     session: myproj
#     path: ~/projects/my-project
#     layout: auto
#     vars:
#       CUSTOM_VAR: value

# Define custom layouts inline (or put in layouts/ directory)
# layouts:
#   my-custom:
#     panes:
#       - cmd: "${EDITOR:-}"
#       - cmd: ""

tools:
  cursor_agent:
    pane_title: cursor
    cmd: ""
  codex_new:
    pane_title: codex
    cmd: ""

# Tool detection + input profiles for CLI/TUI sends
# tool_detection:
#   enabled: true
#   allow:
#     codex: true
#     claude: true
#     lazygit: true
#     gh-dash: true
#   profiles:
#     codex:
#       bracketed_paste: true
#       submit: "\\r"
#       submit_delay_ms: 30
#   tools:
#     - name: my-tool
#       command_regex: ["(?i)mytool"]
#       title_regex: ["(?i)mytool"]
#       input:
#         submit: "\\r"
`

// DefaultGlobalConfigContent returns the default global config template text.
func DefaultGlobalConfigContent() string {
	return defaultGlobalConfigContent
}

// EnsureDefaultGlobalConfig creates the default global config if missing.
func EnsureDefaultGlobalConfig(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is empty")
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("config path %q is a directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat config %q: %w", path, err)
	}
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	layoutsDir := filepath.Join(configDir, identity.GlobalLayoutsDir)
	if err := os.MkdirAll(layoutsDir, 0o755); err != nil {
		return fmt.Errorf("create layouts dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(defaultGlobalConfigContent), 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}
