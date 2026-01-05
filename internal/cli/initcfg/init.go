package initcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/regenrek/peakypanes/internal/cli/output"
	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/layout"
)

// Register registers init handler.
func Register(reg *root.Registry) {
	reg.Register("init", runInit)
}

func runInit(ctx root.CommandContext) error {
	start := time.Now()
	meta := output.NewMeta("init", ctx.Deps.Version)
	local := ctx.Cmd.Bool("local")
	layoutName := strings.TrimSpace(ctx.Cmd.String("layout"))
	if layoutName == "" {
		layoutName = layout.DefaultLayoutName
	}
	force := ctx.Cmd.Bool("force")
	appName := identity.NormalizeCLIName(ctx.Deps.AppName)
	var err error
	if local {
		err = initLocal(appName, layoutName, force)
	} else {
		err = initGlobal(appName, layoutName, force)
	}
	if err != nil {
		return err
	}
	if ctx.JSON {
		meta = output.WithDuration(meta, start)
		return output.WriteSuccess(ctx.Out, meta, output.ActionResult{
			Action: "init",
			Status: "ok",
		})
	}
	return nil
}

func initLocal(appName, layoutName string, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine current directory: %w", err)
	}

	configPath := filepath.Join(cwd, ".peakypanes.yml")
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf(".peakypanes.yml already exists (use --force to overwrite)")
		}
	}

	baseLayout, err := layout.GetBuiltinLayout(layoutName)
	if err != nil {
		return fmt.Errorf("layout %q not found", layoutName)
	}

	projectName := filepath.Base(cwd)
	content := fmt.Sprintf(`# Peaky Panes - Project Layout Configuration
# This file defines the Peaky Panes layout for this project.
# Teammates with %s installed will get this layout automatically.
#
# Variables: ${PROJECT_NAME}, ${PROJECT_PATH}, ${EDITOR}, or any env var
# Use ${VAR:-default} for defaults

session: %s

layout:
`, appName, projectName)

	yamlContent, err := baseLayout.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize layout: %w", err)
	}

	lines := strings.Split(yamlContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "name:") || strings.HasPrefix(line, "description:") {
			continue
		}
		if line != "" {
			content += "  " + line + "\n"
		}
	}

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", configPath, err)
	}

	fmt.Printf("✨ Created %s\n", configPath)
	fmt.Printf("   Based on layout: %s\n", layoutName)
	fmt.Printf("\n   Edit it to customize, then:\n")
	fmt.Printf("   • Run '%s start' to start the session\n", appName)
	fmt.Printf("   • Run '%s' to open the dashboard\n", appName)
	fmt.Printf("   • Commit to git so teammates get the same layout\n")
	return nil
}

func initGlobal(appName, layoutName string, force bool) error {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return fmt.Errorf("cannot determine config path: %w", err)
	}

	layoutsDir, err := layout.DefaultLayoutsDir()
	if err != nil {
		return fmt.Errorf("cannot determine layouts dir: %w", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.MkdirAll(layoutsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create layouts dir: %w", err)
	}

	if !force {
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config already exists: %s\n", configPath)
			fmt.Printf("Use --force to overwrite\n")
			return nil
		}
	}

	configContent := `# Peaky Panes - Global Configuration
# https://github.com/regenrek/peakypanes

zellij:
  # config: ~/.config/zellij/config.kdl
  # layout_dir: ~/.config/zellij/layouts
  # bridge_plugin: ~/.config/peakypanes/zellij/peakypanes-bridge.wasm

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
#   preview_mode: grid   # grid | layout
#   attach_behavior: current  # current | detached
#   project_roots:
#     - ~/projects
#   status_regex:
#     success: "(?i)done|finished|success|completed|✅"
#     error: "(?i)error|failed|panic|❌"
#     running: "(?i)running|in progress|building|installing|▶"
#   agent_detection:
#     codex: true
#     claude: true

# Peky agent settings (used by /peky and Shift+Tab)
# agent:
#   provider: google
#   model: gemini-3-flash
#   # If allowed_commands is set, only these commands may run.
#   # allowed_commands:
#   #   - pane.add
#   #   - pane.split
#   #   - session.kill
#   # Otherwise, blocked_commands denies specific commands/prefixes.
#   blocked_commands:
#     - daemon
#     - daemon.*

# Quick reply settings (for @file picker)
# quick_reply:
#   files:
#     show_hidden: false
#     max_depth: 4
#     max_items: 500

# Load additional layouts from this directory
layout_dirs:
  - ~/.config/peakypanes/layouts

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

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✨ Initialized Peaky Panes!\n\n")
	fmt.Printf("   Config: %s\n", configPath)
	fmt.Printf("   Layouts: %s\n\n", layoutsDir)
	fmt.Printf("   Next steps:\n")
	fmt.Printf("   • Run '%s layouts' to see available layouts\n", appName)
	fmt.Printf("   • Run '%s init --local' in a project to create .peakypanes.yml\n", appName)
	fmt.Printf("   • Run '%s start' in any directory to start a session\n", appName)
	fmt.Printf("   • Run '%s' to open the dashboard\n", appName)
	return nil
}
