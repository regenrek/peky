package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/mux"
	"github.com/regenrek/peakypanes/internal/tmuxctl"
	"github.com/regenrek/peakypanes/internal/tui/peakypanes"
	"github.com/regenrek/peakypanes/internal/zellijctl"
)

var version = "dev"

const (
	defaultDashboardSession = "peakypanes-dashboard"
	defaultDashboardWindow  = "peakypanes-dashboard"
)

const helpText = `🎩 Peaky Panes - Tmux/Zellij Layout Manager

Usage:
  peakypanes [command] [options]

Commands:
  (no command)     Open interactive dashboard (direct)
  dashboard        Open dashboard UI
  popup            Open dashboard as a popup (tmux/zellij)
  open             Start/attach session in current directory
  start            Start/attach session (same as open)
  kill             Kill a tmux/zellij session
  init             Initialize configuration
  layouts          List and manage layouts
  clone            Clone from GitHub and open
	setup            Check dependencies and print install tips
  version          Show version

Examples:
  peakypanes                          # Open dashboard
  peakypanes dashboard                # Open dashboard (direct)
  peakypanes dashboard --tmux-session # Host dashboard in tmux session
  peakypanes popup                    # Open dashboard popup (tmux/zellij)
  peakypanes open                     # Start/attach session in current directory
  peakypanes open --layout dev-3      # Start with specific layout
  peakypanes kill                     # Kill session for current directory
  peakypanes kill myapp               # Kill specific session
  peakypanes init                     # Create global config
  peakypanes init --local             # Create .peakypanes.yml in current dir
  peakypanes layouts                  # List available layouts
  peakypanes layouts export dev-3     # Export layout YAML to stdout
  peakypanes clone user/repo          # Clone from GitHub and start session
	peakypanes setup                    # Check tmux installation

Run 'peakypanes <command> --help' for more information.
`

const dashboardHelpText = `Open the Peaky Panes dashboard UI.

Usage:
  peakypanes dashboard [options]

Options:
  --tmux-session       Host the dashboard in a dedicated tmux/zellij session
  --session <name>     Session name for --tmux-session (default: peakypanes-dashboard)
  --popup              Open the dashboard as a popup when supported
  --mux <name>         Multiplexer to target (tmux or zellij)
  -h, --help           Show this help

Examples:
  peakypanes dashboard
  peakypanes dashboard --tmux-session
  peakypanes dashboard --tmux-session --session my-dashboard
  peakypanes dashboard --popup
`

const popupHelpText = `Open the dashboard as a popup (fallbacks to direct UI).

Usage:
  peakypanes popup [--mux tmux|zellij]
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

const layoutsHelpText = `List and manage layouts.

Usage:
  peakypanes layouts [subcommand]

Subcommands:
  (none)               List all available layouts
  export <name>        Print layout YAML to stdout
  
Options:
  --mux <name>         Multiplexer to target (tmux or zellij)
  -h, --help           Show this help

Examples:
  peakypanes layouts                  # List all layouts
  peakypanes layouts export dev-3     # Print dev-3 layout YAML
  peakypanes layouts export dev-3 > .peakypanes.yml
`

const startHelpText = `Start or attach to a tmux/zellij session.

Usage:
  peakypanes start [options]

Options:
  --layout <name>      Use specific layout (default: auto-detect)
  --session <name>     Override session name (default: directory name)
  --path <dir>         Project directory (default: current directory)
  -d, --detach         Create session but do not attach
  --mux <name>         Multiplexer to use (tmux or zellij)
  -h, --help           Show this help

Layout Detection (in order):
  1. --layout flag
  2. .peakypanes.yml in project directory
  3. Project entry in ~/.config/peakypanes/config.yml
  4. Builtin 'dev-3' layout

Examples:
  peakypanes start                    # Auto-detect layout
  peakypanes start --mux zellij       # Use zellij backend
  peakypanes start --layout fullstack
  peakypanes start --session myapp --layout go-dev
  peakypanes start --detach
`

const killHelpText = `Kill a tmux/zellij session.

Usage:
  peakypanes kill [session-name]

Arguments:
  session-name         Session to kill (default: current directory name)

Options:
  -h, --help           Show this help

Examples:
  peakypanes kill                     # Kill session for current directory
  peakypanes kill myapp               # Kill session named 'myapp'
`

const setupHelpText = `Check external dependencies.

Usage:
	peakypanes setup [--mux tmux|zellij]

Checks:
	tmux and/or zellij is installed and on PATH

Examples:
	peakypanes setup
`

func main() {
	if len(os.Args) < 2 {
		// Default: open dashboard directly in the current terminal
		runMenu()
		return
	}

	switch os.Args[1] {
	case "dashboard", "ui":
		runDashboardCommand(os.Args[2:])
	case "popup":
		runDashboardPopup(os.Args[2:])
	case "open", "o", "start":
		runStart(os.Args[2:])
	case "kill", "k":
		runKill(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "layouts":
		runLayouts(os.Args[2:])
	case "clone", "c":
		runClone(os.Args[2:])
	case "setup":
		runSetup(os.Args[2:])
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

func runMenu() {
	runMenuWithMux("")
}

func runMenuWithMux(cliMux string) {
	cfg, _, err := loadGlobalConfig()
	if err != nil {
		fatal("failed to load config: %v", err)
	}
	muxType := resolveMuxType(cliMux, cfg, nil, nil)
	client, err := newMuxClient(muxType, cfg)
	if err != nil {
		fatal("failed to initialize %s: %v", muxType, err)
	}

	model, err := peakypanes.NewModel(client)
	if err != nil {
		fatal("failed to initialize: %v", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fatal("TUI error: %v", err)
	}
}

type dashboardOptions struct {
	popup      bool
	tmuxHosted bool
	session    string
	showHelp   bool
	mux        string
}

func runDashboardCommand(args []string) {
	opts := dashboardOptions{session: defaultDashboardSession}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tmux-session":
			opts.tmuxHosted = true
		case "--mux", "--multiplexer":
			if i+1 < len(args) {
				opts.mux = args[i+1]
				i++
			}
		case "--session":
			if i+1 < len(args) {
				opts.session = args[i+1]
				i++
			}
		case "--popup":
			opts.popup = true
		case "-h", "--help":
			opts.showHelp = true
		}
	}

	if opts.showHelp {
		fmt.Print(dashboardHelpText)
		return
	}
	if opts.popup && opts.tmuxHosted {
		fatal("choose either --popup or --tmux-session")
	}
	if opts.popup {
		runDashboardPopupWithMux(opts.mux)
		return
	}
	if opts.tmuxHosted {
		runDashboardHosted(opts.session, opts.mux)
		return
	}
	runMenuWithMux(opts.mux)
}

func runDashboardPopup(args []string) {
	cliMux := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Print(popupHelpText)
			return
		case "--mux", "--multiplexer":
			if i+1 < len(args) {
				cliMux = args[i+1]
				i++
			}
		}
	}
	runDashboardPopupWithMux(cliMux)
}

func runDashboardPopupWithMux(cliMux string) {
	cfg, _, err := loadGlobalConfig()
	if err != nil {
		fatal("failed to load config: %v", err)
	}
	muxType := resolveMuxType(cliMux, cfg, nil, nil)
	client, err := newMuxClient(muxType, cfg)
	if err != nil {
		fatal("failed to initialize %s: %v", muxType, err)
	}
	dashboardCmd := []string{selfExecutable(), "dashboard", "--mux", muxType.String()}
	if !client.IsInside() {
		runMenuWithMux(cliMux)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if client.SupportsPopup(ctx) {
		err := client.DisplayPopup(ctx, mux.PopupOptions{
			Width:    "90%",
			Height:   "90%",
			StartDir: currentDir(),
		}, dashboardCmd)
		if err == nil {
			return
		}
	}
	if err := client.OpenDashboardWindow(ctx, "", defaultDashboardWindow, dashboardCmd); err != nil {
		fmt.Fprintf(os.Stderr, "peakypanes: popup failed: %v\n", err)
		runMenuWithMux(cliMux)
	}
}

func runDashboardHosted(sessionName, cliMux string) {
	cfg, _, err := loadGlobalConfig()
	if err != nil {
		fatal("failed to load config: %v", err)
	}
	muxType := resolveMuxType(cliMux, cfg, nil, nil)
	switch muxType {
	case mux.Tmux:
		runDashboardHostedTmux(sessionName)
	case mux.Zellij:
		runDashboardHostedZellij(sessionName, cfg)
	default:
		fatal("unsupported multiplexer: %s", muxType)
	}
}

func runDashboardHostedTmux(sessionName string) {
	client, err := tmuxctl.NewClient("")
	if err != nil {
		fatal("tmux not found: %v", err)
	}
	sessionName = sanitizeSessionName(strings.TrimSpace(sessionName))
	if sessionName == "" {
		sessionName = defaultDashboardSession
	}
	exe := selfExecutable()
	if insideTmux() {
		cmd := exec.Command(client.Binary(), "new-session", "-Ad", "-s", sessionName, exe, "dashboard", "--mux", "tmux")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatal("failed to start dashboard session: %v", err)
		}
		switchCmd := exec.Command(client.Binary(), "switch-client", "-t", sessionName)
		switchCmd.Stdin = os.Stdin
		switchCmd.Stdout = os.Stdout
		switchCmd.Stderr = os.Stderr
		if err := switchCmd.Run(); err != nil {
			fmt.Printf("   Run: tmux switch-client -t %s\n", sessionName)
		}
		return
	}
	cmd := exec.Command(client.Binary(), "new-session", "-A", "-s", sessionName, exe, "dashboard", "--mux", "tmux")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("failed to start dashboard session: %v", err)
	}
}

func runDashboardHostedZellij(sessionName string, cfg *layout.Config) {
	sessionName = sanitizeSessionName(strings.TrimSpace(sessionName))
	if sessionName == "" {
		sessionName = defaultDashboardSession
	}
	bridgePath := ""
	if cfg != nil && strings.TrimSpace(cfg.Zellij.BridgePlugin) != "" {
		bridgePath = expandUserPath(cfg.Zellij.BridgePlugin)
	}
	zClient, err := zellijctl.NewClient("", bridgePath)
	if err != nil {
		fatal("zellij not found: %v", err)
	}
	bridgePath = zClient.BridgePath()
	configPath, err := ensureZellijConfig(cfg, bridgePath)
	if err != nil {
		fatal("failed to build zellij config: %v", err)
	}
	layoutCfg := &layout.LayoutConfig{
		Windows: []layout.WindowDef{{
			Name: defaultDashboardWindow,
			Panes: []layout.PaneDef{{
				Cmd: strings.Join([]string{selfExecutable(), "dashboard", "--mux", "zellij"}, " "),
			}},
		}},
	}
	layoutContent, err := zellijctl.BuildLayout(layoutCfg, currentDir())
	if err != nil {
		fatal("failed to build dashboard layout: %v", err)
	}
	layoutDir, err := resolveZellijLayoutDir(cfg)
	if err != nil {
		fatal("failed to resolve zellij layout dir: %v", err)
	}
	layoutPath, err := zellijctl.WriteLayoutFile(layoutDir, sessionName+"-dashboard", layoutContent)
	if err != nil {
		fatal("failed to write zellij layout: %v", err)
	}

	if insideZellij() {
		args := []string{"action", "new-tab", "--name", defaultDashboardWindow, "--", selfExecutable(), "dashboard", "--mux", "zellij"}
		cmd := exec.Command(zClient.Binary(), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fatal("failed to open dashboard tab: %v", err)
		}
		return
	}

	cmd := exec.Command(zClient.Binary(), "--session", sessionName, "--layout", layoutPath, "--config", configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("failed to start dashboard session: %v", err)
	}
}

func runSetup(args []string) {
	showHelp := false
	cliMux := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			showHelp = true
		case "--mux", "--multiplexer":
			if i+1 < len(args) {
				cliMux = args[i+1]
				i++
			}
		}
	}
	if showHelp {
		fmt.Print(setupHelpText)
		return
	}

	fmt.Println("🎩 Peaky Panes setup")
	fmt.Println()

	cfg, _, err := loadGlobalConfig()
	if err != nil {
		fmt.Printf("⚠️  config: %v\n\n", err)
		cfg = &layout.Config{}
	}

	configuredMux := ""
	if cfg != nil {
		configuredMux = strings.TrimSpace(cfg.Multiplexer)
	}

	if strings.TrimSpace(cliMux) != "" || configuredMux != "" {
		muxType := resolveMuxType(cliMux, cfg, nil, nil)
		ok := checkMuxSetup(muxType)
		if ok {
			fmt.Println()
			fmt.Println("All set.")
			fmt.Println("Run 'peakypanes' to open the dashboard.")
			return
		}
		os.Exit(1)
	}

	tmuxOK := checkMuxSetup(mux.Tmux)
	fmt.Println()
	zellijOK := checkMuxSetup(mux.Zellij)
	fmt.Println()
	if tmuxOK || zellijOK {
		fmt.Println("All set.")
		fmt.Println("Set multiplexer in ~/.config/peakypanes/config.yml to choose a default.")
		fmt.Println("Run 'peakypanes' to open the dashboard.")
		return
	}

	fmt.Println("Install tmux or zellij, then rerun 'peakypanes setup'.")
	os.Exit(1)
}

func checkMuxSetup(muxType mux.Type) bool {
	switch muxType {
	case mux.Tmux:
		return checkTmuxSetup()
	case mux.Zellij:
		return checkZellijSetup()
	default:
		fmt.Printf("Unsupported multiplexer: %s\n", muxType)
		return false
	}
}

func checkTmuxSetup() bool {
	tmuxPath, err := exec.LookPath("tmux")
	if err == nil && strings.TrimSpace(tmuxPath) != "" {
		fmt.Printf("✅ tmux found: %s\n", tmuxPath)
		if out, err := exec.Command("tmux", "-V").Output(); err == nil {
			v := strings.TrimSpace(string(out))
			if v != "" {
				fmt.Printf("   %s\n", v)
			}
		}
		return true
	}

	fmt.Println("❌ tmux not found in PATH")
	fmt.Println()
	fmt.Println("Install tmux, then rerun 'peakypanes setup'.")
	printTmuxInstallTips()
	return false
}

func checkZellijSetup() bool {
	zellijPath, err := exec.LookPath("zellij")
	if err == nil && strings.TrimSpace(zellijPath) != "" {
		fmt.Printf("✅ zellij found: %s\n", zellijPath)
		if out, err := exec.Command("zellij", "--version").Output(); err == nil {
			v := strings.TrimSpace(string(out))
			if v != "" {
				fmt.Printf("   %s\n", v)
			}
		}
		return true
	}

	fmt.Println("❌ zellij not found in PATH")
	fmt.Println()
	fmt.Println("Install zellij, then rerun 'peakypanes setup'.")
	printZellijInstallTips()
	return false
}

func printTmuxInstallTips() {
	switch runtime.GOOS {
	case "darwin":
		fmt.Println()
		fmt.Println("macOS")
		fmt.Println("  brew install tmux")
		fmt.Println("  or")
		fmt.Println("  port install tmux")
	case "linux":
		fmt.Println()
		fmt.Println("Debian or Ubuntu")
		fmt.Println("  sudo apt-get update")
		fmt.Println("  sudo apt-get install tmux")
		fmt.Println()
		fmt.Println("Fedora")
		fmt.Println("  sudo dnf install tmux")
		fmt.Println()
		fmt.Println("Arch")
		fmt.Println("  sudo pacman -S tmux")
	case "windows":
		fmt.Println()
		fmt.Println("Windows")
		fmt.Println("  tmux runs in WSL")
		fmt.Println("  wsl --install")
		fmt.Println("  then inside WSL")
		fmt.Println("    sudo apt-get update")
		fmt.Println("    sudo apt-get install tmux")
	default:
		fmt.Println()
		fmt.Println("Install tmux with your system package manager.")
	}
}

func printZellijInstallTips() {
	switch runtime.GOOS {
	case "darwin":
		fmt.Println()
		fmt.Println("macOS")
		fmt.Println("  brew install zellij")
		fmt.Println("  or")
		fmt.Println("  sudo port install zellij")
	case "linux":
		fmt.Println()
		fmt.Println("Arch")
		fmt.Println("  sudo pacman -S zellij")
		fmt.Println()
		fmt.Println("Fedora")
		fmt.Println("  sudo dnf copr enable varlad/zellij")
		fmt.Println("  sudo dnf install zellij")
		fmt.Println()
		fmt.Println("Void")
		fmt.Println("  sudo xbps-install zellij")
	default:
		fmt.Println()
		fmt.Println("Install zellij with your system package manager.")
	}
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

	// Clone the repository
	cmd := exec.Command("git", "clone", url, targetDir)
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
	local := false
	layoutName := "dev-3"
	force := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local", "-l":
			local = true
		case "--layout":
			if i+1 < len(args) {
				layoutName = args[i+1]
				i++
			}
		case "--force", "-f":
			force = true
		case "-h", "--help":
			fmt.Print(initHelpText)
			return
		}
	}

	if local {
		initLocal(layoutName, force)
	} else {
		initGlobal(layoutName, force)
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
# This file defines the tmux/zellij layout for this project.
# Teammates with peakypanes installed will get this layout automatically.
# Optional: set multiplexer to "tmux" or "zellij".
# multiplexer: tmux
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

multiplexer: tmux

tmux:
  config: ~/.config/tmux/tmux.conf

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
#   project_roots:
#     - ~/projects
#   status_regex:
#     success: "(?i)done|finished|success|completed|✅"
#     error: "(?i)error|failed|panic|❌"
#     running: "(?i)running|in progress|building|installing|▶"

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
#     windows:
#       - name: dev
#         panes:
#           - title: editor
#             cmd: "${EDITOR:-}"
#           - title: shell
#             cmd: ""

tools:
  cursor_agent:
    window_name: cursor
    cmd: ""
  codex_new:
    window_name: codex
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

func runKill(args []string) {
	sessionName := ""
	cliMux := ""
	var projectPath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Print(killHelpText)
			return
		case "--mux", "--multiplexer":
			if i+1 < len(args) {
				cliMux = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && sessionName == "" {
				sessionName = args[i]
			}
		}
	}

	// Default session name to current directory name
	if sessionName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fatal("cannot determine current directory: %v", err)
		}
		projectPath = cwd
		sessionName = sanitizeSessionName(filepath.Base(cwd))
	} else {
		cwd, err := os.Getwd()
		if err == nil {
			projectPath = cwd
		}
	}

	cfg, _, err := loadGlobalConfig()
	if err != nil {
		fatal("failed to load config: %v", err)
	}

	var localCfg *layout.ProjectLocalConfig
	if projectPath != "" {
		if cfg, err := layout.LoadProjectLocal(projectPath); err == nil {
			localCfg = cfg
		}
	}
	projectCfg := findProjectConfig(cfg, projectPath, sessionName)
	muxType := resolveMuxType(cliMux, cfg, projectCfg, localCfg)
	client, err := newMuxClient(muxType, cfg)
	if err != nil {
		fatal("failed to initialize %s: %v", muxType, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if session exists
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		fatal("failed to list sessions: %v", err)
	}

	found := false
	for _, s := range sessions {
		if s == sessionName {
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ Session '%s' not found\n", sessionName)
		if len(sessions) > 0 {
			fmt.Printf("\n   Running sessions:\n")
			for _, s := range sessions {
				fmt.Printf("   • %s\n", s)
			}
		}
		return
	}

	// Kill the session
	if err := client.KillSession(ctx, sessionName); err != nil {
		fatal("failed to kill session: %v", err)
	}

	fmt.Printf("✅ Killed session '%s'\n", sessionName)
}

func runStart(args []string) {
	layoutName := ""
	sessionName := ""
	projectPath := ""
	detach := false
	cliMux := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--layout", "-l":
			if i+1 < len(args) {
				layoutName = args[i+1]
				i++
			}
		case "--session", "-s":
			if i+1 < len(args) {
				sessionName = args[i+1]
				i++
			}
		case "--path", "-p":
			if i+1 < len(args) {
				projectPath = args[i+1]
				i++
			}
		case "--detach", "-d":
			detach = true
		case "--mux", "--multiplexer":
			if i+1 < len(args) {
				cliMux = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Print(startHelpText)
			return
		default:
			// Treat as layout name shortcut if not a flag
			if !strings.HasPrefix(args[i], "-") && layoutName == "" {
				layoutName = args[i]
			}
		}
	}

	// Default to current directory
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			fatal("cannot determine current directory: %v", err)
		}
	}

	cfg, _, err := loadGlobalConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "peakypanes: config: %v\n", err)
		cfg = &layout.Config{}
	}

	// Load layouts
	loader, err := layout.NewLoader()
	if err != nil {
		fatal("failed to create loader: %v", err)
	}
	loader.SetProjectDir(projectPath)

	if err := loader.LoadAll(); err != nil {
		fatal("failed to load layouts: %v", err)
	}

	// Load session name from config if not explicitly provided via flag
	localCfg := loader.GetProjectConfig()
	if sessionName == "" && localCfg != nil && localCfg.Session != "" {
		sessionName = localCfg.Session
	}

	// Default to directory name if still empty
	if sessionName == "" {
		sessionName = sanitizeSessionName(filepath.Base(projectPath))
	}

	// Determine which layout to use
	var selectedLayout *layout.LayoutConfig
	var source string

	if layoutName != "" {
		// Explicit layout requested
		selectedLayout, source, err = loader.GetLayout(layoutName)
		if err != nil {
			fatal("layout %q not found. Run 'peakypanes layouts' to see available layouts.", layoutName)
		}
	} else if loader.HasProjectConfig() {
		// Use project-local config
		selectedLayout = loader.GetProjectLayout()
		source = "project"
		if selectedLayout == nil {
			// Project config exists but no layout defined, use default
			selectedLayout, source, _ = loader.GetLayout("dev-3")
		}
	} else {
		// Fall back to default
		selectedLayout, source, _ = loader.GetLayout("dev-3")
	}

	if selectedLayout == nil {
		fatal("no layout found")
	}

	// Expand variables
	projectName := filepath.Base(projectPath)
	var projectVars map[string]string
	if localCfg != nil {
		projectVars = localCfg.Vars
	}
	expandedLayout := layout.ExpandLayoutVars(selectedLayout, projectVars, projectPath, projectName)

	projectCfg := findProjectConfig(cfg, projectPath, sessionName)
	muxType := resolveMuxType(cliMux, cfg, projectCfg, localCfg)
	switch muxType {
	case mux.Tmux:
		runStartTmux(sessionName, projectPath, expandedLayout, source, detach)
	case mux.Zellij:
		runStartZellij(sessionName, projectPath, expandedLayout, source, detach, cfg)
	default:
		fatal("unsupported multiplexer: %s", muxType)
	}
}

func runStartTmux(sessionName, projectPath string, layoutCfg *layout.LayoutConfig, source string, detach bool) {
	// Create tmux client
	client, err := tmuxctl.NewClient("")
	if err != nil {
		fatal("tmux not found: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	maybeSourceTmuxConfig(ctx, client)

	// Check if session already exists
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		fatal("failed to list sessions: %v", err)
	}

	sessionExists := false
	for _, s := range sessions {
		if s == sessionName {
			sessionExists = true
			break
		}
	}

	fmt.Printf("🎩 Peaky Panes\n")
	fmt.Printf("   Session: %s\n", sessionName)
	fmt.Printf("   Layout:  %s (%s)\n", layoutCfg.Name, source)
	fmt.Printf("   Path:    %s\n", projectPath)
	fmt.Println()

	if sessionExists {
		applyLayoutBindings(ctx, client, layoutCfg)
		if detach {
			fmt.Printf("   Session already exists.\n\n")
			fmt.Printf("   Leaving session detached.\n")
			return
		}
		fmt.Printf("   Session already exists, attaching...\n\n")
		attachToSession(client, sessionName)
		return
	}

	// Create the session with layout
	fmt.Println("   Creating windows:")
	if err := createSessionWithLayout(ctx, client, sessionName, projectPath, layoutCfg); err != nil {
		fatal("failed to create session: %v", err)
	}

	applyLayoutBindings(ctx, client, layoutCfg)

	fmt.Println()
	fmt.Printf("   ✅ Session created!\n\n")

	// Attach to session
	if detach {
		fmt.Printf("   Leaving session detached.\n")
		return
	}
	attachToSession(client, sessionName)
}

func runStartZellij(sessionName, projectPath string, layoutCfg *layout.LayoutConfig, source string, detach bool, cfg *layout.Config) {
	bridgePath := ""
	if cfg != nil && strings.TrimSpace(cfg.Zellij.BridgePlugin) != "" {
		bridgePath = expandUserPath(cfg.Zellij.BridgePlugin)
	}
	zClient, err := zellijctl.NewClient("", bridgePath)
	if err != nil {
		fatal("zellij not found: %v", err)
	}
	bridgePath = zClient.BridgePath()
	configPath, err := ensureZellijConfig(cfg, bridgePath)
	if err != nil {
		fatal("failed to build zellij config: %v", err)
	}

	layoutContent, err := zellijctl.BuildLayout(layoutCfg, projectPath)
	if err != nil {
		fatal("failed to build layout: %v", err)
	}
	layoutDir, err := resolveZellijLayoutDir(cfg)
	if err != nil {
		fatal("failed to resolve zellij layout dir: %v", err)
	}
	layoutPath, err := zellijctl.WriteLayoutFile(layoutDir, sessionName, layoutContent)
	if err != nil {
		fatal("failed to write zellij layout: %v", err)
	}
	_ = zellijctl.RecordSessionPath(sessionName, projectPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessions, err := zClient.ListSessions(ctx)
	if err != nil {
		fatal("failed to list sessions: %v", err)
	}

	sessionExists := false
	for _, s := range sessions {
		if s == sessionName {
			sessionExists = true
			break
		}
	}

	fmt.Printf("🎩 Peaky Panes\n")
	fmt.Printf("   Session: %s\n", sessionName)
	fmt.Printf("   Layout:  %s (%s)\n", layoutCfg.Name, source)
	fmt.Printf("   Path:    %s\n", projectPath)
	fmt.Println()

	if sessionExists {
		if detach {
			fmt.Printf("   Session already exists.\n\n")
			fmt.Printf("   Leaving session detached.\n")
			return
		}
		fmt.Printf("   Session already exists, attaching...\n\n")
		if insideZellij() {
			if err := zClient.SwitchSession(ctx, "", sessionName, nil); err != nil {
				fatal("failed to switch session: %v", err)
			}
			return
		}
		if err := zClient.AttachSession(ctx, sessionName); err != nil {
			fatal("failed to attach session: %v", err)
		}
		return
	}

	fmt.Println("   Creating session:")
	if detach {
		if err := startZellijDetached(zClient, sessionName, layoutPath, configPath); err != nil {
			fatal("failed to create session: %v", err)
		}
		fmt.Println()
		fmt.Printf("   ✅ Session created!\n\n")
		fmt.Printf("   Leaving session detached.\n")
		return
	}

	if insideZellij() {
		if err := startZellijDetached(zClient, sessionName, layoutPath, configPath); err != nil {
			fatal("failed to create session: %v", err)
		}
		if err := waitForZellijSession(ctx, zClient, sessionName); err != nil {
			fatal("failed to start session: %v", err)
		}
		if err := zClient.SwitchSession(ctx, "", sessionName, nil); err != nil {
			fatal("failed to switch session: %v", err)
		}
		return
	}

	if err := startZellijAttached(zClient, sessionName, layoutPath, configPath); err != nil {
		fatal("failed to start session: %v", err)
	}
}

func startZellijAttached(client *zellijctl.Client, sessionName, layoutPath, configPath string) error {
	args := []string{}
	if strings.TrimSpace(configPath) != "" {
		args = append(args, "--config", configPath)
	}
	if strings.TrimSpace(layoutPath) != "" {
		args = append(args, "--layout", layoutPath)
	}
	args = append(args, "--session", sessionName)
	cmd := exec.Command(client.Binary(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startZellijDetached(client *zellijctl.Client, sessionName, layoutPath, configPath string) error {
	args := []string{}
	if strings.TrimSpace(configPath) != "" {
		args = append(args, "--config", configPath)
	}
	if strings.TrimSpace(layoutPath) != "" {
		args = append(args, "--layout", layoutPath)
	}
	args = append(args, "attach", sessionName, "--create-background")
	cmd := exec.Command(client.Binary(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForZellijSession(ctx context.Context, client *zellijctl.Client, sessionName string) error {
	deadline, hasDeadline := ctx.Deadline()
	for {
		sessions, err := client.ListSessions(ctx)
		if err == nil {
			for _, s := range sessions {
				if s == sessionName {
					return nil
				}
			}
		}
		if hasDeadline && time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("session %q not available yet", sessionName)
}

func createSessionWithLayout(ctx context.Context, client *tmuxctl.Client, session, projectPath string, layoutCfg *layout.LayoutConfig) error {
	if strings.TrimSpace(layoutCfg.Grid) != "" {
		return createSessionWithGridLayout(ctx, client, session, projectPath, layoutCfg)
	}
	if len(layoutCfg.Windows) == 0 {
		return fmt.Errorf("layout has no windows defined")
	}

	// Create first window with session
	firstWindow := layoutCfg.Windows[0]
	firstCmd := ""
	if len(firstWindow.Panes) > 0 && firstWindow.Panes[0].Cmd != "" {
		firstCmd = firstWindow.Panes[0].Cmd
	}

	firstPaneID, err := client.NewSessionWithCmd(ctx, session, projectPath, firstWindow.Name, firstCmd)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Give tmux a moment to register the new session
	time.Sleep(200 * time.Millisecond)

	// Apply peakypanes default tmux options (session-scoped, not global)
	// remain-on-exit: on keeps panes open when commands exit, which is better for debugging
	_ = client.SetOption(ctx, session, "remain-on-exit", "on")

	// Apply custom tmux options from layout config
	for option, value := range layoutCfg.Settings.TmuxOptions {
		_ = client.SetOption(ctx, session, option, value)
	}

	// Set title for first pane
	if len(firstWindow.Panes) > 0 && firstWindow.Panes[0].Title != "" {
		_ = client.SelectPane(ctx, firstPaneID, firstWindow.Panes[0].Title)
	}

	fmt.Printf("   • %s ", firstWindow.Name)

	// Create additional panes in first window
	for i := 1; i < len(firstWindow.Panes); i++ {
		pane := firstWindow.Panes[i]
		vertical := pane.Split == "vertical" || pane.Split == "v"

		// Parse size percentage
		percent := 0
		if pane.Size != "" {
			sizeStr := strings.TrimSuffix(pane.Size, "%")
			if p, err := strconv.Atoi(sizeStr); err == nil {
				percent = p
			}
		}

		// Target the window directly
		target := fmt.Sprintf("%s:%s", session, firstWindow.Name)

		// Give tmux a tiny moment
		time.Sleep(100 * time.Millisecond)

		newPaneID, err := client.SplitWindowWithCmd(ctx, target, projectPath, vertical, percent, pane.Cmd)
		if err != nil {
			return fmt.Errorf("split pane: %w", err)
		}

		if pane.Title != "" {
			_ = client.SelectPane(ctx, newPaneID, pane.Title)
		}
	}

	fmt.Printf("(%d panes)\n", len(firstWindow.Panes))

	// Apply layout if specified (after all panes are created)
	if firstWindow.Layout != "" {
		windowTarget := fmt.Sprintf("%s:%s", session, firstWindow.Name)
		if err := client.SelectLayout(ctx, windowTarget, firstWindow.Layout); err != nil {
			fmt.Printf("   ⚠ Layout %s: %v\n", firstWindow.Layout, err)
		}
	}

	// Create additional windows
	for _, win := range layoutCfg.Windows[1:] {
		firstCmd := ""
		if len(win.Panes) > 0 && win.Panes[0].Cmd != "" {
			firstCmd = win.Panes[0].Cmd
		}

		firstPaneID, err := client.NewWindowWithCmd(ctx, session, win.Name, projectPath, firstCmd)
		if err != nil {
			return fmt.Errorf("create window %s: %w", win.Name, err)
		}

		if len(win.Panes) > 0 && win.Panes[0].Title != "" {
			_ = client.SelectPane(ctx, firstPaneID, win.Panes[0].Title)
		}

		fmt.Printf("   • %s ", win.Name)

		// Create additional panes
		for i := 1; i < len(win.Panes); i++ {
			pane := win.Panes[i]
			vertical := pane.Split == "vertical" || pane.Split == "v"

			percent := 0
			if pane.Size != "" {
				sizeStr := strings.TrimSuffix(pane.Size, "%")
				if p, err := strconv.Atoi(sizeStr); err == nil {
					percent = p
				}
			}

			// Target the window directly
			target := fmt.Sprintf("%s:%s", session, win.Name)

			// Give tmux a tiny moment
			time.Sleep(100 * time.Millisecond)

			newPaneID, err := client.SplitWindowWithCmd(ctx, target, projectPath, vertical, percent, pane.Cmd)
			if err != nil {
				return fmt.Errorf("split pane in %s: %w", win.Name, err)
			}

			if pane.Title != "" {
				_ = client.SelectPane(ctx, newPaneID, pane.Title)
			}
		}

		// Apply layout if specified
		if win.Layout != "" {
			windowTarget := fmt.Sprintf("%s:%s", session, win.Name)
			_ = client.SelectLayout(ctx, windowTarget, win.Layout)
		}

		fmt.Printf("(%d panes)\n", len(win.Panes))
	}

	// Select first window and first pane
	if len(layoutCfg.Windows) > 0 {
		windowTarget := fmt.Sprintf("%s:%s", session, layoutCfg.Windows[0].Name)
		// Select window
		_ = exec.CommandContext(ctx, "tmux", "select-window", "-t", windowTarget).Run()
		// Select first pane
		_ = exec.CommandContext(ctx, "tmux", "select-pane", "-t", windowTarget+".0").Run()
	}

	return nil
}

func createSessionWithGridLayout(ctx context.Context, client *tmuxctl.Client, session, projectPath string, layoutCfg *layout.LayoutConfig) error {
	grid, err := layout.Parse(layoutCfg.Grid)
	if err != nil {
		return fmt.Errorf("parse grid %q: %w", layoutCfg.Grid, err)
	}

	windowName := strings.TrimSpace(layoutCfg.Window)
	if windowName == "" {
		windowName = strings.TrimSpace(layoutCfg.Name)
	}
	if windowName == "" {
		windowName = "grid"
	}

	firstPaneID, err := client.NewSessionWithCmd(ctx, session, projectPath, windowName, "")
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	time.Sleep(200 * time.Millisecond)

	_ = client.SetOption(ctx, session, "remain-on-exit", "on")
	for option, value := range layoutCfg.Settings.TmuxOptions {
		_ = client.SetOption(ctx, session, option, value)
	}

	fmt.Printf("   • %s ", windowName)

	if grid.Columns > 1 {
		if err := splitGridColumns(ctx, client, projectPath, firstPaneID, grid.Columns); err != nil {
			return err
		}
	}

	windowTarget := fmt.Sprintf("%s:%s", session, windowName)

	columnPanes, err := client.ListPanesDetailed(ctx, windowTarget)
	if err != nil {
		return fmt.Errorf("list grid columns: %w", err)
	}
	sort.SliceStable(columnPanes, func(i, j int) bool {
		return columnPanes[i].Left < columnPanes[j].Left
	})
	if len(columnPanes) < grid.Columns {
		return fmt.Errorf("expected %d columns, found %d panes", grid.Columns, len(columnPanes))
	}
	columnPanes = columnPanes[:grid.Columns]

	if grid.Rows > 1 {
		for _, pane := range columnPanes {
			if err := splitGridRows(ctx, client, projectPath, pane.ID, grid.Rows); err != nil {
				return err
			}
		}
	}

	panes, err := client.ListPanesDetailed(ctx, windowTarget)
	if err != nil {
		return fmt.Errorf("list grid panes: %w", err)
	}
	sort.SliceStable(panes, func(i, j int) bool {
		if panes[i].Top == panes[j].Top {
			return panes[i].Left < panes[j].Left
		}
		return panes[i].Top < panes[j].Top
	})
	if len(panes) < grid.Panes() {
		return fmt.Errorf("expected %d panes, found %d panes", grid.Panes(), len(panes))
	}

	commands := resolveGridCommands(layoutCfg, grid.Panes())
	titles := resolveGridTitles(layoutCfg, grid.Panes())

	for i := 0; i < grid.Panes(); i++ {
		pane := panes[i]
		if strings.TrimSpace(titles[i]) != "" {
			_ = client.SelectPane(ctx, pane.ID, titles[i])
		}
		if strings.TrimSpace(commands[i]) != "" {
			_ = client.SendKeys(ctx, pane.ID, commands[i], "C-m")
		}
	}

	fmt.Printf("(%d panes)\n", grid.Panes())

	windowTarget = fmt.Sprintf("%s:%s", session, windowName)
	_ = exec.CommandContext(ctx, "tmux", "select-window", "-t", windowTarget).Run()
	_ = exec.CommandContext(ctx, "tmux", "select-pane", "-t", windowTarget+".0").Run()

	return nil
}

func splitGridColumns(ctx context.Context, client *tmuxctl.Client, projectPath, targetPane string, columns int) error {
	remaining := columns
	for remaining > 1 {
		percent := gridSplitPercent(remaining)
		if _, err := client.SplitWindowWithCmd(ctx, targetPane, projectPath, false, percent, ""); err != nil {
			return fmt.Errorf("split grid columns: %w", err)
		}
		remaining--
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

func splitGridRows(ctx context.Context, client *tmuxctl.Client, projectPath, targetPane string, rows int) error {
	remaining := rows
	for remaining > 1 {
		percent := gridSplitPercent(remaining)
		if _, err := client.SplitWindowWithCmd(ctx, targetPane, projectPath, true, percent, ""); err != nil {
			return fmt.Errorf("split grid rows: %w", err)
		}
		remaining--
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

func gridSplitPercent(remaining int) int {
	if remaining <= 1 {
		return 0
	}
	percent := int(math.Round(100.0 / float64(remaining)))
	if percent < 1 {
		return 1
	}
	if percent >= 100 {
		return 99
	}
	return percent
}

func resolveGridCommands(layoutCfg *layout.LayoutConfig, count int) []string {
	commands := make([]string, 0, count)
	fallback := strings.TrimSpace(layoutCfg.Command)
	if len(layoutCfg.Commands) > 0 {
		for i := 0; i < count; i++ {
			if i < len(layoutCfg.Commands) {
				commands = append(commands, layoutCfg.Commands[i])
				continue
			}
			if fallback != "" {
				commands = append(commands, fallback)
			} else {
				commands = append(commands, "")
			}
		}
		return commands
	}
	if fallback == "" {
		for i := 0; i < count; i++ {
			commands = append(commands, "")
		}
		return commands
	}
	for i := 0; i < count; i++ {
		commands = append(commands, fallback)
	}
	return commands
}

func resolveGridTitles(layoutCfg *layout.LayoutConfig, count int) []string {
	titles := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if i < len(layoutCfg.Titles) {
			titles = append(titles, layoutCfg.Titles[i])
		} else {
			titles = append(titles, "")
		}
	}
	return titles
}

func applyLayoutBindings(ctx context.Context, client *tmuxctl.Client, layoutCfg *layout.LayoutConfig) {
	if layoutCfg == nil {
		return
	}
	for _, bind := range layoutCfg.Settings.BindKeys {
		if strings.TrimSpace(bind.Key) == "" || strings.TrimSpace(bind.Action) == "" {
			continue
		}
		if err := client.BindKey(ctx, bind.Key, bind.Action); err != nil {
			fmt.Printf("   ⚠ bind-key %s: %v\n", bind.Key, err)
		}
	}
}

func maybeSourceTmuxConfig(ctx context.Context, client *tmuxctl.Client) {
	configPath, err := layout.DefaultConfigPath()
	if err != nil {
		return
	}
	if _, err := os.Stat(configPath); err != nil {
		return
	}
	cfg, err := layout.LoadConfig(configPath)
	if err != nil || cfg == nil {
		return
	}
	tmuxConfig := expandUserPath(cfg.Tmux.Config)
	if strings.TrimSpace(tmuxConfig) == "" {
		return
	}
	if _, err := os.Stat(tmuxConfig); err != nil {
		return
	}
	if err := client.SourceFile(ctx, tmuxConfig); err != nil {
		fmt.Printf("   ⚠ tmux config: %v\n", err)
	}
}

func expandUserPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	path = os.ExpandEnv(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func attachToSession(client *tmuxctl.Client, session string) {
	// Check if we're inside tmux
	if os.Getenv("TMUX") != "" {
		// Switch client
		cmd := exec.Command("tmux", "switch-client", "-t", session)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Printf("   Run: tmux switch-client -t %s\n", session)
		}
	} else {
		// Attach session
		cmd := exec.Command("tmux", "attach-session", "-t", session)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Printf("   Run: tmux attach -t %s\n", session)
		}
	}
}

func sanitizeSessionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "session"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "session"
	}
	return result
}

func insideTmux() bool {
	return os.Getenv("TMUX") != "" || os.Getenv("TMUX_PANE") != ""
}

func insideZellij() bool {
	return strings.TrimSpace(os.Getenv("ZELLIJ")) != "" || strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME")) != ""
}

func selfExecutable() string {
	exe, err := os.Executable()
	if err != nil || strings.TrimSpace(exe) == "" {
		return "peakypanes"
	}
	return exe
}

func currentDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func openDashboardWindow(ctx context.Context, client *tmuxctl.Client) error {
	session, err := client.CurrentSession(ctx)
	if err != nil {
		return err
	}
	if session == "" {
		return fmt.Errorf("no active tmux session")
	}
	windows, err := client.ListWindows(ctx, session)
	if err != nil {
		return err
	}
	for _, w := range windows {
		if w.Name == defaultDashboardWindow {
			return client.SelectWindow(ctx, fmt.Sprintf("%s:%s", session, defaultDashboardWindow))
		}
	}
	cmd := exec.Command(client.Binary(), "new-window", "-t", session, "-n", defaultDashboardWindow, selfExecutable(), "dashboard")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return client.SelectWindow(ctx, fmt.Sprintf("%s:%s", session, defaultDashboardWindow))
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "peakypanes: "+format+"\n", args...)
	os.Exit(1)
}
