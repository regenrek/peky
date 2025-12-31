package main

const helpText = `🎩 Peaky Panes - Native Session Dashboard

Usage:
  peakypanes [command] [options]

Commands:
  (no command)     Open interactive dashboard
  dashboard        Open dashboard UI
  open             Start a session and open dashboard
  start            Start a session and open dashboard (alias)
  daemon           Run or manage the session daemon
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
  peakypanes daemon restart           # Restart daemon in background
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
  --fresh-config       Start with no global config or saved state
  --temporary-run      Use a temporary runtime + config dir (implies --fresh-config)

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
  peakypanes daemon [subcommand] [options]

Subcommands:
  (none)           Run daemon in foreground
  restart          Restart daemon and restore sessions

Options:
  -h, --help           Show this help

Examples:
  peakypanes daemon
  peakypanes daemon restart --yes
`

const daemonRestartHelpText = `Restart the Peaky Panes session daemon.

Usage:
  peakypanes daemon restart [options]

Options:
  -y, --yes            Skip confirmation prompt
  -h, --help           Show this help

Examples:
  peakypanes daemon restart
  peakypanes daemon restart --yes
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
  --fresh-config       Start with no global config or saved state
  --temporary-run      Use a temporary runtime + config dir (implies --fresh-config)
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
