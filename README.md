# 🎩 Peaky Panes

**Tmux layout manager with YAML-based configuration.**

Define your tmux layouts in YAML, share them with your team via git, and get consistent development environments everywhere.

## Features

- 📦 **Built-in layouts** - Works out of the box with sensible defaults
- 📁 **Project-local config** - Commit `.peakypanes.yml` to git for team sharing
- 🏠 **Global config** - Define layouts once, use everywhere
- 🔄 **Variable expansion** - Use `${EDITOR}`, `${PROJECT_PATH}`, etc.
- 🎯 **Zero config** - Just run `peakypanes` in any directory

## Quick Start

### Install

```bash
go install github.com/kregenrek/tmuxman/cmd/peakypanes@latest
```

### Usage

**Just run it:**
```bash
cd your-project
peakypanes
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

## Built-in Layouts

| Layout | Description |
|--------|-------------|
| `simple` | Single window, one pane |
| `split-v` | Two vertical panes (left/right) |
| `split-h` | Two horizontal panes (top/bottom) |
| `dev-2` | Editor + terminal |
| `dev-3` | Editor + server + shell (default) |
| `fullstack` | Editor + dev server + logs window |
| `go-dev` | Editor + run + tests + lazygit |
| `tauri-debug` | Complex Tauri/Rust development layout |

```bash
# List all layouts
peakypanes layouts

# Export a layout to customize
peakypanes layouts export dev-3 > .peakypanes.yml
```

## Configuration

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
```

### Global Config (`~/.config/peakypanes/config.yml`)

For personal layouts and multi-project management:

```yaml
# Global settings
tmux:
  config: ~/.config/tmux/tmux.conf

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
peakypanes                     # Start session (auto-detect layout)
peakypanes start               # Same as above
peakypanes start --layout X    # Use specific layout
peakypanes init                # Create global config
peakypanes init --local        # Create .peakypanes.yml
peakypanes layouts             # List available layouts
peakypanes layouts export X    # Export layout YAML
peakypanes version             # Show version
```

## How Layout Detection Works

1. `--layout` flag (highest priority)
2. `.peakypanes.yml` in current directory
3. Project entry in `~/.config/peakypanes/config.yml`
4. Built-in `dev-3` layout (fallback)

## For Teams

1. Run `peakypanes init --local` in your project
2. Customize `.peakypanes.yml` for your stack
3. Commit to git
4. Teammates install peakypanes and run `peakypanes` - done!

## License

MIT
