package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/layout"
)

type initOptions struct {
	local    bool
	layout   string
	force    bool
	showHelp bool
}

func runInit(args []string) {
	opts := parseInitArgs(args)
	if opts.showHelp {
		fmt.Print(initHelpText)
		return
	}

	if opts.local {
		initLocal(opts.layout, opts.force)
	} else {
		initGlobal(opts.layout, opts.force)
	}
}

func parseInitArgs(args []string) initOptions {
	opts := initOptions{layout: "dev-3"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local", "-l":
			opts.local = true
		case "--layout":
			if i+1 < len(args) {
				opts.layout = args[i+1]
				i++
			}
		case "--force", "-f":
			opts.force = true
		case "-h", "--help":
			opts.showHelp = true
		}
	}
	return opts
}

func initLocal(layoutName string, force bool) {
	cwd, err := os.Getwd()
	if err != nil {
		fatal("cannot determine current directory: %v", err)
	}

	configPath := filepath.Join(cwd, ".peakypanes.yml")
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			fatal(".peakypanes.yml already exists (use --force to overwrite)")
		}
	}

	baseLayout, err := layout.GetBuiltinLayout(layoutName)
	if err != nil {
		fatal("layout %q not found", layoutName)
	}

	projectName := filepath.Base(cwd)
	content := fmt.Sprintf(`# Peaky Panes - Project Layout Configuration
# This file defines the Peaky Panes layout for this project.
# Teammates with peakypanes installed will get this layout automatically.
#
# Variables: ${PROJECT_NAME}, ${PROJECT_PATH}, ${EDITOR}, or any env var
# Use ${VAR:-default} for defaults

session: %s

layout:
`, projectName)

	yamlContent, err := baseLayout.ToYAML()
	if err != nil {
		fatal("failed to serialize layout: %v", err)
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
		fatal("failed to write %s: %v", configPath, err)
	}

	fmt.Printf("✨ Created %s\n", configPath)
	fmt.Printf("   Based on layout: %s\n", layoutName)
	fmt.Printf("\n   Edit it to customize, then:\n")
	fmt.Printf("   • Run 'peakypanes start' to start the session\n")
	fmt.Printf("   • Run 'peakypanes' to open the dashboard\n")
	fmt.Printf("   • Commit to git so teammates get the same layout\n")
}

func initGlobal(layoutName string, force bool) {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		fatal("cannot determine config path: %v", err)
	}

	layoutsDir, err := layout.DefaultLayoutsDir()
	if err != nil {
		fatal("cannot determine layouts dir: %v", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fatal("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(layoutsDir, 0o755); err != nil {
		fatal("failed to create layouts dir: %v", err)
	}

	if !force {
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config already exists: %s\n", configPath)
			fmt.Printf("Use --force to overwrite\n")
			return
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

# Load additional layouts from this directory
layout_dirs:
  - ~/.config/peakypanes/layouts

# Define projects for quick access
# projects:
#   - name: my-project
#     session: myproj
#     path: ~/projects/my-project
#     layout: dev-3
#     vars:
#       CUSTOM_VAR: value

# Define custom layouts inline (or put in layouts/ directory)
# layouts:
#   my-custom:
#     panes:
#       - title: editor
#         cmd: "${EDITOR:-}"
#       - title: shell
#         cmd: ""

tools:
  cursor_agent:
    pane_title: cursor
    cmd: ""
  codex_new:
    pane_title: codex
    cmd: ""
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		fatal("failed to write config: %v", err)
	}

	fmt.Printf("✨ Initialized Peaky Panes!\n\n")
	fmt.Printf("   Config: %s\n", configPath)
	fmt.Printf("   Layouts: %s\n\n", layoutsDir)
	fmt.Printf("   Next steps:\n")
	fmt.Printf("   • Run 'peakypanes layouts' to see available layouts\n")
	fmt.Printf("   • Run 'peakypanes init --local' in a project to create .peakypanes.yml\n")
	fmt.Printf("   • Run 'peakypanes start' in any directory to start a session\n")
	fmt.Printf("   • Run 'peakypanes' to open the dashboard\n")
}
