# 🎩 Peaky Panes

```
████    █████    ███    █   █   █   █    ████      ███    █   █    █████    ████
█████   ████    █████   ████     ███     █████    █████   ███ █    ████    ████ 
█       █████   █   █   █  ██     █      █        █   █   █  ██    █████   █████
```

**Tmux layout manager with YAML-based configuration.**

![Peaky Panes Preview](assets/peakypanes-preview.jpg)


Define your tmux layouts in YAML, share them with your team via git, and get consistent development environments everywhere.

## Features

- 📦 **Built-in layouts** - Works out of the box with sensible defaults
- 🧩 **Exact grids** - Use `grid: 2x3` for consistent rows/columns
- 📁 **Project-local config** - Commit `.peakypanes.yml` to git for team sharing
- 🏠 **Global config** - Define layouts once, use everywhere
- 🔄 **Variable expansion** - Use `${EDITOR}`, `${PROJECT_PATH}`, etc.
- 🎯 **Zero config** - Just run `peakypanes` in any directory
- ⚙️ **Session-scoped tmux options** - Configure tmux per-session without affecting global config
- 🪟 **Popup dashboard** - Open the UI as a tmux popup when available
- ⌘ **Command palette** - Quick actions, including renaming sessions/windows

## Quick Start

### Install

**Using npm (recommended)**

```bash
npm i -g peakypanes
peakypanes
```

> [!TIP]
> Run `peakypanes setup` to check dependencies

**Run once with npx**

```bash
npx -y peakypanes
```

Using Go

```bash
go install github.com/regenrek/peakypanes/cmd/peakypanes@latest
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

## Configuration

> 📖 **[Layout Builder Guide](docs/layout-builder.md)** - Detailed documentation on creating custom layouts, pane arrangements, and tmux options.

### Project-Local (`.peakypanes.yml`)

Create in your project root for team-shared layouts:

```yaml
# .peakypanes.yml
session: my-project

layout:
  windows:
    - name: dev
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

    - name: logs
      panes:
        - title: docker
          cmd: "docker compose logs -f"

# Or use exact grids
# layout:
#   grid: 2x3
#   window: codex
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
# Global settings
tmux:
  # Optional: source a custom tmux config when starting sessions.
  # (tmux already reads ~/.tmux.conf or ~/.config/tmux/tmux.conf by default)
  config: ~/.config/tmux/tmux.conf

# Dashboard UI settings (optional)
# dashboard:
#   project_roots:
#     - ~/projects
#     - ~/code

# Custom layouts
layouts:
  my-custom:
    windows:
      - name: main
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
  windows:
    - name: dev
      panes:
        - cmd: "tail -f ${log_file}"
```

## Commands

```bash
peakypanes                     # Open dashboard (direct)
peakypanes dashboard           # Open dashboard (direct)
peakypanes dashboard --tmux-session  # Host dashboard in tmux session
peakypanes dashboard --popup   # Open dashboard as a tmux popup
peakypanes popup               # Open dashboard as a tmux popup
peakypanes open                # Start/attach session in current directory
peakypanes start               # Same as open
peakypanes start --layout X    # Use specific layout
peakypanes start --detach      # Create session without attaching
peakypanes kill [session]      # Kill a tmux session
peakypanes init                # Create global config
peakypanes init --local        # Create .peakypanes.yml
peakypanes layouts             # List available layouts
peakypanes layouts export X    # Export layout YAML
peakypanes clone user/repo     # Clone from GitHub and start session
peakypanes setup               # Check external dependencies
peakypanes version             # Show version
```

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
- `fullstack`: dev + logs
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
Use `peakypanes dashboard --tmux-session` to host the dashboard in a dedicated tmux session.
Use `peakypanes popup` (or `peakypanes dashboard --popup`) from inside tmux for a popup dashboard.
If popups are unsupported, PeakyPanes opens a `peakypanes-dashboard` window in the current tmux session.

The dashboard shows:
- Projects on top (tabs)
- Sessions on the left (with window counts and expandable windows)
- Live pane preview on the right (window bar at the bottom)
- Lightweight session thumbnails at the bottom (last activity per session)
- Quick reply bar (always visible) and target pane highlight for follow-ups

Navigation (always visible):
- `ctrl+a/ctrl+d` project, `ctrl+w/ctrl+s` session, `tab/⇧tab` pane (across windows), `ctrl+g` help

Key bindings (also shown in the help view):
Keymap overrides are available in the global config (`~/.config/peakypanes/config.yml`).

Project
- `ctrl+o` open project picker (creates session detached; stay in dashboard)
- `ctrl+b` close project (hides from tabs; sessions keep running; press k in the dialog to kill)

Session
- `enter` attach/start session (when reply is empty)
- `ctrl+n` new session (pick layout)
- `ctrl+t` open in new terminal window
- `ctrl+x` kill session
- rename session via command palette (`ctrl+p`)

Window
- `ctrl+u` toggle window list
- rename window via command palette (`ctrl+p`)

Pane
- rename pane via command palette (`ctrl+p`)
- `ctrl+y` peek selected pane in new terminal

Tmux (inside session)
- `prefix+g` open dashboard popup (tmux prefix is yours)

Other
- `ctrl+p` command palette
- `ctrl+r` refresh, `ctrl+e` edit config, `ctrl+f` filter, `ctrl+c` quit

Quick reply details: the input is always active—type and press `enter` to send to the highlighted pane. Use `esc` to clear. `tab/⇧tab` still cycles panes while the input is focused.

### Dashboard Config (optional)

```yaml
dashboard:
  refresh_ms: 2000
  preview_lines: 12
  preview_compact: true
  thumbnail_lines: 1
  idle_seconds: 20
  show_thumbnails: true
  preview_mode: grid  # grid | layout
  attach_behavior: new_terminal  # current | new_terminal | detached
  keymap:
    project_left: ["ctrl+a"]
    project_right: ["ctrl+d"]
    session_up: ["ctrl+w"]
    session_down: ["ctrl+s"]
    pane_next: ["tab"]
    pane_prev: ["shift+tab"]
    peek_pane: ["ctrl+y"]
    toggle_windows: ["ctrl+u"]
    command_palette: ["ctrl+p"]
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

`attach_behavior` controls what the “attach/start” action does (default `new_terminal`): `current` switches the terminal running PeakyPanes into the session, `new_terminal` opens a fresh terminal to attach, and `detached` only creates the session.

### Agent Status Detection (Codex & Claude Code)

PeakyPanes can read per-pane JSON state files to show accurate running/idle/done status for Codex CLI and Claude Code TUI sessions. This is **on by default** and falls back to regex/idle detection if no state file is present. You can disable it via `dashboard.agent_detection`.

State files are written under `${XDG_RUNTIME_DIR:-/tmp}/peakypanes/agent-state` and keyed by `TMUX_PANE` (override with `PEAKYPANES_AGENT_STATE_DIR`).

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

### Tmux Config & Key Bindings

- PeakyPanes **never edits** your tmux config file.
- tmux already reads `~/.tmux.conf` or `~/.config/tmux/tmux.conf` by default.
- If you use a **custom tmux config path**, set `tmux.config` in `~/.config/peakypanes/config.yml`.
  PeakyPanes will **source** that file when starting sessions (no overwrite).
- Per-layout tmux options and key bindings are supported:

```yaml
settings:
  tmux_options:
    remain-on-exit: "on"
  bind_keys:
    - key: g
      action: "run-shell \"peakypanes popup\""
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

Tmux integration tests (requires tmux; opt-in):

```bash
PEAKYPANES_INTEGRATION=1 go test ./internal/tmuxctl -run Integration -count=1
```

Manual npm smoke run (fresh HOME/XDG config):

```bash
scripts/fresh-run
scripts/fresh-run 0.0.2 --with-project
```

GitHub Actions runs gofmt checks, go vet, go test with coverage, race, and tmux integration tests on Linux.

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
