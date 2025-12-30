# 🎩 Peaky Panes

```
████    █████    ███    █   █   █   █    ████      ███    █   █    █████    ████
█████   ████    █████   ████     ███     █████    █████   ███ █    ████    ████ 
█       █████   █   █   █  ██     █      █        █   █   █  ██    █████   █████
```

**Terminal dashboard with YAML-based layouts, native live previews, and persistent native sessions.**

![Peaky Panes Preview](assets/preview-peakypanes-v2.jpg)


Define your layouts in YAML, share them with your team via git, and get consistent development environments everywhere. Sessions are owned by a **native daemon** so they keep running after the UI exits.

## Features

- 📦 **Built-in layouts** - Works out of the box with sensible defaults
- 🧩 **Exact grids** - Use `grid: 2x3` for consistent rows/columns
- 📁 **Project-local config** - Commit `.peakypanes.yml` to git for team sharing
- 🏠 **Global config** - Define layouts once, use everywhere
- 🔄 **Variable expansion** - Use `${EDITOR}`, `${PROJECT_PATH}`, etc.
- 🎯 **Zero config** - Just run `peakypanes` in any directory
- 🧠 **Native live previews** - Full TUI support (vim/htop) with live panes
- 🧭 **Persistent native daemon** - Sessions keep running after the UI exits
- 📜 **Scrollback + copy mode** - Navigate output and yank from native panes
- ⌘ **Command palette** - Quick actions, including renaming sessions/panes
- 🖱️ **Mouse support** - Click to select panes, double-click to focus a pane

## Quick Start

### Install

**Using npm (recommended)**

```bash
npm i -g peakypanes
peakypanes
```

**Run once with npx**

```bash
npx -y peakypanes
```

**Using Homebrew**

```bash
brew tap regenrek/tap
brew install regenrek/tap/peakypanes
```

Using Go

```bash
go install github.com/regenrek/peakypanes/cmd/peakypanes@latest
```

**Hot reload (from repo)**

```bash
scripts/dev-watch -- --layout dev-3
```

### Usage

**Start a session (auto-detect layout):**
```bash
peakypanes start
```

**Use a specific layout:**
```bash
peakypanes start --layout dev-3
peakypanes start --layout fullstack
```

**Create project-local config (recommended for teams):**
```bash
cd your-project
peakypanes init --local
# Edit .peakypanes.yml
git add .peakypanes.yml  # Share with team
```

**Run the daemon in the foreground (optional):**
```bash
peakypanes daemon
```

## Configuration

> 📖 **[Layout Builder Guide](docs/layout-builder.md)** - Detailed documentation on creating custom layouts, pane arrangements, and configuration.

### Project-Local (`.peakypanes.yml`)

Create in your project root for team-shared layouts:

```yaml
# .peakypanes.yml
session: my-project

layout:
  panes:
    - title: editor
      cmd: "${EDITOR:-}"
      size: "60%"
    - title: server
      cmd: "npm run dev"
      split: horizontal
    - title: shell
      cmd: ""
      split: vertical
    - title: docker
      cmd: "docker compose logs -f"

# Or use exact grids
# layout:
#   grid: 2x3
#   commands:
#     - "${SHELL:-bash}"
#     - "codex"
#     - "codex"
#     - "codex"
#     - "codex"
#     - "codex"
#   titles:
#     - shell
#     - codex-1
#     - codex-2
#     - codex-3
#     - codex-4
#     - codex-5
```

### Global Config (`~/.config/peakypanes/config.yml`)

For personal layouts and multi-project management:

```yaml
# Dashboard UI settings (optional)
# dashboard:
#   project_roots:
#     - ~/projects
#     - ~/code

# Custom layouts
layouts:
  my-custom:
    panes:
      - title: code
        cmd: nvim
      - title: term
        cmd: ""

# Projects for quick switching
projects:
  - name: webapp
    session: webapp
    path: ~/projects/webapp
    layout: fullstack
```

## Variable Expansion

Use variables in your layouts:

| Variable | Description |
|----------|-------------|
| `${PROJECT_PATH}` | Absolute path to project |
| `${PROJECT_NAME}` | Directory name |
| `${EDITOR}` | Your $EDITOR |
| `${VAR:-default}` | Env var with default |

```yaml
layout:
  vars:
    log_file: "${HOME}/logs/${PROJECT_NAME}.log"
  panes:
    - cmd: "tail -f ${log_file}"
```

## Commands

```bash
peakypanes                     # Open dashboard (direct)
peakypanes dashboard           # Open dashboard (direct)
peakypanes open                # Start session and open dashboard
peakypanes start               # Same as open
peakypanes start --layout X    # Use specific layout
peakypanes init                # Create global config
peakypanes init --local        # Create .peakypanes.yml
peakypanes layouts             # List available layouts
peakypanes layouts export X    # Export layout YAML
peakypanes clone user/repo     # Clone from GitHub and start session
peakypanes version             # Show version
```

## Troubleshooting: daemon stuck / restart

The daemon owns sessions and PTYs. If it becomes unresponsive, the only recovery
today is a manual restart, which **will terminate all running sessions**.

**Manual restart (macOS default path):**
```bash
kill "$(cat "$HOME/Library/Application Support/peakypanes/daemon.pid")"
```

**Manual restart (Linux default path):**
```bash
kill "$(cat "$HOME/.config/peakypanes/daemon.pid")"
```

You can also set `PEAKYPANES_DAEMON_PID` to control the pid file location.

### Proposed UX (future)
If the app detects a hung daemon, it should show a dialog like:
**“Restart daemon? This will stop all running sessions and close their PTYs.”**
This makes the data-loss tradeoff explicit before taking action.

## Built-in Layouts

Core (general) layouts:
- `auto` (default): no layout flag; auto-detects `.peakypanes.yml` or falls back to `dev-3`
- `simple`: single pane
- `split-v`: two vertical panes (left/right)
- `split-h`: two horizontal panes (top/bottom)
- `2x2`: 4‑pane grid
- `3x4`: 12‑pane grid
- `codex-dev`: 2x3 grid (shell + 5 codex)

Additional built-ins (specialized):
- `dev-2`: editor + shell
- `dev-3`: editor + server + shell (default fallback)
- `fullstack`: editor + server + shell + logs
- `go-dev`: code/run/test + git
- `codex-grid`: 2x4 grid running codex in every pane

```bash
# List all layouts
peakypanes layouts

# Export a layout to customize
peakypanes layouts export codex-dev > .peakypanes.yml
```

## Dashboard UI

Running `peakypanes` with no subcommand opens the dashboard UI in the current terminal.

The dashboard shows:
- Projects on top (tabs)
- Sessions on the left (with pane counts and expandable panes)
- Live pane preview on the right (native panes are fully interactive)
- Input bar (always visible) and target pane highlight for follow-ups

Navigation (always visible):
- `ctrl+q/ctrl+e` project, `ctrl+w/ctrl+s` session/panes, `alt+w/alt+s` session only, `tab/⇧tab` or `ctrl+a/ctrl+d` pane, `ctrl+g` help

Key bindings (also shown in the help view):
Keymap overrides are available in the global config (`~/.config/peakypanes/config.yml`).

Project
- `ctrl+o` open project picker (creates session detached; stay in dashboard)
- `ctrl+b` close project (hides from tabs; sessions keep running; press k in the dialog to kill)

Session
- `enter` attach/start session (when reply is empty)
- `ctrl+n` new session (pick layout)
- `ctrl+x` kill session
- rename session via command palette (`ctrl+p`)

Window
Pane list
- `ctrl+u` toggle pane list

Pane
- rename pane via command palette (`ctrl+p`)
- `ctrl+y` peek selected pane in new terminal
- `ctrl+\` toggle terminal focus (native only; configurable via `dashboard.keymap.terminal_focus`)
- mouse: single-click selects a pane; double-click toggles terminal focus (native only); `esc` exits focus
- `f7` scrollback mode (native only; configurable via `dashboard.keymap.scrollback`)
- `f8` copy mode (native only; configurable via `dashboard.keymap.copy_mode`)

Other
- `ctrl+p` command palette
- `ctrl+r` refresh, `ctrl+,` edit config, `ctrl+f` filter, `ctrl+c` quit

Input details: the input is always active—type and press `enter` to send to the highlighted pane. Use `esc` to clear. Toggle terminal focus to send raw keystrokes into the pane. Use scrollback (`f7`) to navigate output and copy mode (`f8`) to select/yank (`v` select, `y` yank, `esc/q` exit).

### Dashboard Config (optional)

```yaml
dashboard:
  refresh_ms: 2000
  preview_lines: 12
  preview_compact: true
  idle_seconds: 20
  preview_mode: grid  # grid | layout
  attach_behavior: current  # current | detached
  keymap:
    project_left: ["ctrl+q"]
    project_right: ["ctrl+e"]
    session_up: ["ctrl+w"]
    session_down: ["ctrl+s"]
    session_only_up: ["alt+w"]
    session_only_down: ["alt+s"]
    pane_next: ["tab", "ctrl+d"]
    pane_prev: ["shift+tab", "ctrl+a"]
    terminal_focus: ["ctrl+\\"]
    scrollback: ["f7"]
    copy_mode: ["f8"]
    toggle_panes: ["ctrl+u"]
    command_palette: ["ctrl+p"]
    edit_config: ["ctrl+,"]
    help: ["ctrl+g"]
    quit: ["ctrl+c"]
  status_regex:
    success: "(?i)done|finished|success|completed|✅"
    error: "(?i)error|failed|panic|❌"
    running: "(?i)running|in progress|building|installing|▶"
  agent_detection:
    codex: true
    claude: true
```

`attach_behavior` controls what the “attach/start” action does (default `current`): `current` focuses the selected session in the dashboard, and `detached` creates the session without switching focus.

### Agent Status Detection (Codex & Claude Code)

PeakyPanes can read per-pane JSON state files to show accurate running/idle/done status for Codex CLI and Claude Code TUI sessions. This is **on by default** and falls back to regex/idle detection if no state file is present. You can disable it via `dashboard.agent_detection`.

State files are written under `${XDG_RUNTIME_DIR:-/tmp}/peakypanes/agent-state` and keyed by `PEAKYPANES_PANE_ID` (override with `PEAKYPANES_AGENT_STATE_DIR`).

**Codex CLI (TUI)**

Add a `notify` command in your Codex config to call the PeakyPanes hook script (Codex passes one JSON arg):

```toml
# ~/.codex/config.toml
notify = ["python3", "/absolute/path/to/peakypanes/scripts/agent-state/codex-notify.py"]
```

Tip: with `npm i -g peakypanes`, the scripts live under `$(npm root -g)/peakypanes/scripts/agent-state/`.
Note: Codex `notify` only fires on turn completion, so running state still relies on regex/idle detection between turns.

**Claude Code (TUI)**

Configure hooks to run the PeakyPanes hook script (Claude passes JSON on stdin). Recommended events:
`SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PermissionRequest`, `Stop`, `SessionEnd`.

Example hook command (wire it to each event above in Claude Code):

```bash
python3 /absolute/path/to/peakypanes/scripts/agent-state/claude-hook.py
```

## How Layout Detection Works

1. `--layout` flag (highest priority)
2. `.peakypanes.yml` in current directory
3. Project entry in `~/.config/peakypanes/config.yml`
4. Built-in `dev-3` layout (fallback)

## Testing

Run the unit tests with coverage:

```bash
go test ./... -coverprofile /tmp/peakypanes.cover
go tool cover -func /tmp/peakypanes.cover | tail -n 1
```

Race tests:

```bash
go test ./... -race
```

Manual npm smoke run (fresh HOME/XDG config):

```bash
scripts/fresh-run
scripts/fresh-run 0.0.5 --with-project
```

GitHub Actions runs gofmt checks, go vet, go test with coverage, and race on Linux.

## Release

See `RELEASE-DOCS.md` for the full release checklist (tests, tag, GoReleaser, npm publish).

## Windows
> npm packages are currently published for macOS and Linux.  
> Windows users should install from the GitHub release or build with Go.


## For Teams

1. Run `peakypanes init --local` in your project
2. Customize `.peakypanes.yml` for your stack
3. Commit to git
4. Teammates install peakypanes and run `peakypanes` - done!

## License

MIT


## Links

- X/Twitter: [@kregenrek](https://x.com/kregenrek)
- Bluesky: [@kevinkern.dev](https://bsky.app/profile/kevinkern.dev)

## Courses
- Learn Cursor AI: [Ultimate Cursor Course](https://www.instructa.ai/en/cursor-ai)
- Learn to build software with AI: [AI Builder Hub](https://www.instructa.ai)

## See my other projects:

* [codefetch](https://github.com/regenrek/codefetch) - Turn code into Markdown for LLMs with one simple terminal command
* [instructa](https://github.com/orgs/instructa/repositories) - Instructa Projects
