package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/peakypanes"
)

var version = "dev"

var (
	runMenuFn = runMenu
	fatalFn   = fatal
)

const helpText = `🎩 Peaky Panes - Native Session Dashboard

Usage:
  peakypanes [command] [options]

Commands:
  (no command)     Open interactive dashboard
  dashboard        Open dashboard UI
  open             Start a session and open dashboard
  start            Start a session and open dashboard (alias)
  daemon           Run the session daemon
  init             Initialize configuration
  layouts          List and manage layouts
  clone            Clone from GitHub and open
  version          Show version

Examples:
  peakypanes                          # Open dashboard
  peakypanes dashboard                # Open dashboard
  peakypanes start                    # Start session from current dir
  peakypanes start --layout dev-3     # Start with specific layout
  peakypanes open --path ~/repo       # Start from a path
  peakypanes daemon                   # Run daemon in foreground
  peakypanes init                     # Create global config
  peakypanes init --local             # Create .peakypanes.yml in current dir
  peakypanes layouts                  # List available layouts
  peakypanes layouts export dev-3     # Export layout YAML to stdout
  peakypanes clone user/repo          # Clone from GitHub and open

Run 'peakypanes <command> --help' for more information.
`

const dashboardHelpText = `Open the Peaky Panes dashboard UI.

Usage:
  peakypanes dashboard [options]

Options:
  -h, --help           Show this help

Examples:
  peakypanes dashboard
`

const initHelpText = `Initialize Peaky Panes configuration.

Usage:
  peakypanes init [options]

Options:
  --local              Create .peakypanes.yml in current directory
  --layout <name>      Start from a template layout (default: dev-3)
  --force              Overwrite existing config
  -h, --help           Show this help

Examples:
  peakypanes init                     # Create ~/.config/peakypanes/
  peakypanes init --local             # Create .peakypanes.yml here
  peakypanes init --local --layout tauri-debug
`

const daemonHelpText = `Run the Peaky Panes session daemon.

Usage:
  peakypanes daemon [options]

Options:
  -h, --help           Show this help

Examples:
  peakypanes daemon
`

const layoutsHelpText = `List and manage layouts.

Usage:
  peakypanes layouts [subcommand]

Subcommands:
  (none)               List all available layouts
  export <name>        Print layout YAML to stdout
  
Options:
  -h, --help           Show this help

Examples:
  peakypanes layouts                  # List all layouts
  peakypanes layouts export dev-3     # Print dev-3 layout YAML
  peakypanes layouts export dev-3 > .peakypanes.yml
`

const startHelpText = `Start a session and open the dashboard.

Usage:
  peakypanes start [options]

Options:
  --layout <name>      Use specific layout (default: auto-detect)
  --session <name>     Override session name (default: directory name)
  --path <dir>         Project directory (default: current directory)
  -h, --help           Show this help

Layout Detection (in order):
  1. --layout flag
  2. .peakypanes.yml in project directory
  3. Project entry in ~/.config/peakypanes/config.yml
  4. Builtin 'dev-3' layout

Examples:
  peakypanes start                    # Auto-detect layout
  peakypanes start --layout fullstack
  peakypanes start --session myapp --layout go-dev
`

func main() {
	if len(os.Args) < 2 {
		// Default: open dashboard directly in the current terminal
		runMenuFn(nil)
		return
	}

	switch os.Args[1] {
	case "dashboard", "ui":
		runDashboardCommand(os.Args[2:])
	case "open", "o", "start":
		runStart(os.Args[2:])
	case "daemon":
		runDaemon(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "layouts":
		runLayouts(os.Args[2:])
	case "clone", "c":
		runClone(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("peakypanes %s\n", version)
	case "help", "-h", "--help":
		fmt.Print(helpText)
	default:
		// Unknown command, could be a layout name shortcut for open
		if !strings.HasPrefix(os.Args[1], "-") {
			runStart(os.Args[1:])
		} else {
			fmt.Print(helpText)
		}
	}
}

func runMenu(autoStart *peakypanes.AutoStartSpec) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := sessiond.ConnectDefault(ctx, version)
	if err != nil {
		fatal("failed to connect to daemon: %v", err)
	}
	defer func() { _ = client.Close() }()

	model, err := peakypanes.NewModel(client)
	if err != nil {
		fatal("failed to initialize: %v", err)
	}
	if autoStart != nil {
		model.SetAutoStart(*autoStart)
	}

	input, cleanup, err := openTUIInput()
	if err != nil {
		fatal("cannot initialize TUI input: %v", err)
	}
	defer cleanup()

	motionFilter := peakypanes.NewMouseMotionFilter()
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithInput(input),
		tea.WithMouseAllMotion(),
		tea.WithFilter(motionFilter.Filter),
	)
	if _, err := p.Run(); err != nil {
		fatal("TUI error: %v", err)
	}
}

func openTUIInput() (*os.File, func(), error) {
	if f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		if err := ensureBlocking(f); err != nil {
			_ = f.Close()
			return nil, func() {}, fmt.Errorf("configure /dev/tty: %w", err)
		}
		return f, func() { _ = f.Close() }, nil
	}
	if err := ensureBlocking(os.Stdin); err != nil {
		return nil, func() {}, fmt.Errorf("stdin is not a usable TUI input: %w", err)
	}
	return os.Stdin, func() {}, nil
}

type dashboardOptions struct {
	showHelp bool
}

type startOptions struct {
	layoutName string
	session    string
	path       string
	showHelp   bool
}

type initOptions struct {
	local    bool
	layout   string
	force    bool
	showHelp bool
}

func runDashboardCommand(args []string) {
	opts := dashboardOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			opts.showHelp = true
		}
	}

	if opts.showHelp {
		fmt.Print(dashboardHelpText)
		return
	}
	runMenuFn(nil)
}

func runDaemon(args []string) {
	showHelp := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			showHelp = true
		}
	}
	if showHelp {
		fmt.Print(daemonHelpText)
		return
	}

	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:       version,
		HandleSignals: true,
	})
	if err != nil {
		fatal("failed to create daemon: %v", err)
	}
	if err := daemon.Run(); err != nil {
		fatal("daemon failed: %v", err)
	}
}

func runStart(args []string) {
	opts := parseStartArgs(args)
	if opts.showHelp {
		fmt.Print(startHelpText)
		return
	}
	projectPath := strings.TrimSpace(opts.path)
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			fatal("cannot determine current directory: %v", err)
		}
	}
	autoStart := &peakypanes.AutoStartSpec{
		Session: opts.session,
		Path:    projectPath,
		Layout:  opts.layoutName,
		Focus:   true,
	}
	runMenuFn(autoStart)
}

func runClone(args []string) {
	if len(args) == 0 {
		fatal("usage: peakypanes clone <url|user/repo>")
	}

	url := args[0]
	// Expand shorthand (user/repo -> https://github.com/user/repo)
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") {
		url = "https://github.com/" + url
	}

	// Extract repo name for directory
	repoName := extractRepoName(url)
	if repoName == "" {
		repoName = "repo"
	}

	// Clone to ~/projects/<repo>
	home, _ := os.UserHomeDir()
	targetDir := filepath.Join(home, "projects", repoName)

	// Check if directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("📁 Directory exists: %s\n", targetDir)
		fmt.Printf("   Starting session...\n\n")
		// Just start the session
		runStartWithPath(targetDir)
		return
	}

	fmt.Printf("🔄 Cloning %s...\n", url)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", url, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("clone failed: %v", err)
	}

	fmt.Printf("\n✅ Cloned to %s\n\n", targetDir)

	// Start session in the cloned directory
	runStartWithPath(targetDir)
}

func runStartWithPath(projectPath string) {
	// Change to the project directory and run start
	origArgs := os.Args
	os.Args = []string{"peakypanes", "start", "--path", projectPath}
	runStart([]string{"--path", projectPath})
	os.Args = origArgs
}

func extractRepoName(url string) string {
	// Handle various URL formats
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
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

	// Get the template layout
	baseLayout, err := layout.GetBuiltinLayout(layoutName)
	if err != nil {
		fatal("layout %q not found", layoutName)
	}

	// Create project config
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

	// Add the layout content (indented)
	yamlContent, err := baseLayout.ToYAML()
	if err != nil {
		fatal("failed to serialize layout: %v", err)
	}

	// Parse and re-marshal just the relevant parts
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

	// Create directories
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		fatal("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(layoutsDir, 0o755); err != nil {
		fatal("failed to create layouts dir: %v", err)
	}

	// Create config file
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

func runLayouts(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "export":
			if len(args) < 2 {
				fatal("usage: peakypanes layouts export <name>")
			}
			exportLayout(args[1])
			return
		case "-h", "--help":
			fmt.Print(layoutsHelpText)
			return
		}
	}

	listLayouts()
}

func listLayouts() {
	loader, err := layout.NewLoader()
	if err != nil {
		fatal("failed to create loader: %v", err)
	}

	cwd, _ := os.Getwd()
	loader.SetProjectDir(cwd)

	if err := loader.LoadAll(); err != nil {
		fatal("failed to load layouts: %v", err)
	}

	layouts := loader.ListLayouts()
	if len(layouts) == 0 {
		fmt.Println("No layouts found.")
		return
	}

	fmt.Println("🎩 Available Layouts")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tDESCRIPTION")
	fmt.Fprintln(w, "----\t------\t-----------")

	for _, l := range layouts {
		source := l.Source
		switch source {
		case "builtin":
			source = "📦 builtin"
		case "global":
			source = "🏠 global"
		case "project":
			source = "📁 project"
		}
		desc := l.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, source, desc)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Use 'peakypanes layouts export <name>' to view layout YAML")
}

func exportLayout(name string) {
	loader, err := layout.NewLoader()
	if err != nil {
		fatal("failed to create loader: %v", err)
	}

	if err := loader.LoadAll(); err != nil {
		fatal("failed to load layouts: %v", err)
	}

	yaml, err := loader.ExportLayout(name)
	if err != nil {
		fatal("layout %q not found", name)
	}

	// Add header comment for project-local use
	fmt.Printf("# Peaky Panes Layout: %s\n", name)
	fmt.Printf("# Save as .peakypanes.yml in your project root\n")
	fmt.Printf("# session: your-session-name  # uncomment to set session name\n\n")
	fmt.Print(yaml)
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

func parseStartArgs(args []string) startOptions {
	opts := startOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--layout", "-l":
			if i+1 < len(args) {
				opts.layoutName = args[i+1]
				i++
			}
		case "--session", "-s":
			if i+1 < len(args) {
				opts.session = args[i+1]
				i++
			}
		case "--path", "-p":
			if i+1 < len(args) {
				opts.path = args[i+1]
				i++
			}
		case "-h", "--help":
			opts.showHelp = true
		default:
			if !strings.HasPrefix(args[i], "-") && opts.layoutName == "" {
				opts.layoutName = args[i]
			}
		}
	}
	return opts
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "peakypanes: "+format+"\n", args...)
	os.Exit(1)
}
