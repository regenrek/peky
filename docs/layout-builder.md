# Layout Builder Guide

This guide covers everything you need to know about creating custom layouts in Peaky Panes, including pane arrangements, tmux options, and advanced configuration.

## Table of Contents

- [Basic Structure](#basic-structure)
- [Pane Layouts](#pane-layouts)
- [Split Directions](#split-directions)
- [Tmux Options](#tmux-options)
- [Variables](#variables)
- [Multi-Window Layouts](#multi-window-layouts)
- [Examples](#examples)

---

## Basic Structure

A `.peakypanes.yml` file has this structure:

```yaml
# Optional: Custom session name (defaults to directory name)
session: my-project

layout:
  name: my-layout
  description: "Description of what this layout is for"
  
  vars:
    # Custom variables
    log_file: "${HOME}/logs/${PROJECT_NAME}.log"
  
  settings:
    width: 240          # Terminal width hint
    height: 84          # Terminal height hint
    tmux_options:       # Session-scoped tmux options
      history-limit: "50000"
  
  windows:
    - name: dev
      layout: tiled     # tmux layout algorithm
      panes:
        - title: editor
          cmd: "${EDITOR:-}"
```

---

## Pane Layouts

### Automatic Layouts (Recommended)

Use the `layout` field on a window to let tmux arrange panes automatically:

| Layout | Description |
|--------|-------------|
| `tiled` | Equal-sized grid (best for 4 panes = 2x2) |
| `even-horizontal` | Side-by-side columns |
| `even-vertical` | Stacked rows |
| `main-horizontal` | Large pane on top, others below |
| `main-vertical` | Large pane on left, others on right |

For exact row/column grids (like 2x3), use the top-level `grid` configuration
instead of `layout: tiled`.

```yaml
windows:
  - name: dev
    layout: tiled      # Automatic 2x2 grid with 4 panes
    panes:
      - title: top-left
        cmd: ""
      - title: top-right
        cmd: ""
      - title: bottom-left
        cmd: ""
      - title: bottom-right
        cmd: ""
```

### Exact Grid Example (2x2)

Use `grid` when you need predictable rows/columns:

```yaml
grid: 2x2
window: main
commands:
  - "codex"
  - "npm run dev"
  - "tail -f app.log"
  - ""
titles:
  - codex
  - dev
  - logs
  - shell
```

---

## Split Directions

For precise control, use `split` on individual panes:

| Split | Result |
|-------|--------|
| `horizontal` | Creates left/right panes |
| `vertical` | Creates top/bottom panes |

```yaml
windows:
  - name: dev
    panes:
      - title: editor        # First pane (no split)
        cmd: "${EDITOR:-}"
        size: "60%"
      - title: server        # Splits horizontally from editor
        cmd: "npm run dev"
        split: horizontal
      - title: shell         # Splits vertically from server
        cmd: ""
        split: vertical
```

This creates:
```
+------------------+----------+
|                  |  server  |
|     editor       +----------+
|                  |  shell   |
+------------------+----------+
```

### Size Control

Use `size` to control pane proportions:

```yaml
panes:
  - title: main
    cmd: ""
    size: "70%"           # Takes 70% of space
  - title: side
    cmd: ""
    split: horizontal
    size: "30%"           # Remaining 30%
```

---

## Tmux Options

Peaky Panes applies session-scoped tmux options that don't affect your global config.

### Default Options

These are applied automatically to all peakypanes sessions:

| Option | Default | Description |
|--------|---------|-------------|
| `remain-on-exit` | `on` | Keeps panes open after command exits/crashes |

### Custom Options

Add your own tmux options per-layout:

```yaml
layout:
  settings:
    tmux_options:
      remain-on-exit: "on"       # Keep crashed panes visible
      history-limit: "50000"     # Scrollback buffer size
      mouse: "on"                # Enable mouse support
      status-position: "top"     # Status bar position
```

### Disabling Defaults

Override defaults if needed:

```yaml
layout:
  settings:
    tmux_options:
      remain-on-exit: "off"      # Let panes close on exit
```

### Key Bindings

Add custom key bindings for the session:

```yaml
layout:
  settings:
    bind_keys:
      - key: "S-Left"
        action: "previous-window"
      - key: "S-Right"
        action: "next-window"
```

---

## Variables

### Built-in Variables

| Variable | Description |
|----------|-------------|
| `${PROJECT_PATH}` | Absolute path to project directory |
| `${PROJECT_NAME}` | Directory name |
| `${HOME}` | User's home directory |

### Environment Variables

Use any environment variable with optional defaults:

```yaml
panes:
  - title: editor
    cmd: "${EDITOR:-vim}"      # Use $EDITOR, fall back to vim
  - title: shell
    cmd: "${SHELL:-/bin/bash}" # Use $SHELL, fall back to bash
```

### Custom Variables

Define reusable variables in the `vars` section:

```yaml
layout:
  vars:
    rust_log: "${HOME}/Library/Logs/${PROJECT_NAME}/rust.log"
    codex_log: "${HOME}/.spezi/codex/log/app-server.log"
    
  windows:
    - name: logs
      panes:
        - title: rust
          cmd: "tail -F ${rust_log}"
        - title: codex
          cmd: "tail -F ${codex_log}"
```

---

## Multi-Window Layouts

Create multiple tmux windows (tabs):

```yaml
layout:
  windows:
    # Window 1: Development
    - name: dev
      layout: tiled
      panes:
        - title: editor
          cmd: "${EDITOR:-}"
        - title: server
          cmd: "npm run dev"
        - title: test
          cmd: ""
        - title: shell
          cmd: ""
    
    # Window 2: Logs
    - name: logs
      layout: even-horizontal
      panes:
        - title: app-log
          cmd: "tail -f logs/app.log"
        - title: error-log
          cmd: "tail -f logs/error.log"
    
    # Window 3: Database
    - name: db
      panes:
        - title: psql
          cmd: "psql -d mydb"
```

---

## Examples

### Full-Stack Web Development

```yaml
session: webapp

layout:
  name: fullstack
  description: "Full-stack development with logs"
  
  settings:
    tmux_options:
      history-limit: "50000"
  
  windows:
    - name: code
      layout: main-vertical
      panes:
        - title: editor
          cmd: "${EDITOR:-}"
        - title: server
          cmd: "npm run dev"
        - title: shell
          cmd: ""
    
    - name: logs
      layout: even-horizontal
      panes:
        - title: frontend
          cmd: "tail -f logs/frontend.log"
        - title: backend
          cmd: "tail -f logs/backend.log"
```

### Tauri/Rust Development

```yaml
session: tauri-app

layout:
  name: tauri-debug
  description: "Tauri development with codex agents"
  
  vars:
    rust_log: "${HOME}/Library/Logs/${PROJECT_NAME}/rust.log"
    codex_log: "${HOME}/.spezi/codex/log/app-server.log"
  
  settings:
    width: 240
    height: 84
    tmux_options:
      remain-on-exit: "on"
  
  windows:
    - name: dev
      layout: tiled
      panes:
        - title: codex
          cmd: "RUST_LOG=debug codex"
        - title: bun-dev
          cmd: "bun dev:tauri"
        - title: codex-logs
          cmd: "tail -F ${codex_log} | grep -Ev '\\bINFO\\b'"
        - title: rust-logs
          cmd: "tail -F ${rust_log}"
```

### Go Development

```yaml
session: go-project

layout:
  name: go-dev
  
  windows:
    - name: dev
      panes:
        - title: editor
          cmd: "${EDITOR:-}"
          size: "60%"
        - title: run
          cmd: ""
          split: horizontal
        - title: test
          cmd: ""
          split: vertical
    
    - name: git
      panes:
        - title: lazygit
          cmd: "lazygit"
```

### Simple 2-Pane Layout

```yaml
layout:
  windows:
    - name: dev
      panes:
        - title: editor
          cmd: "${EDITOR:-}"
          size: "60%"
        - title: terminal
          cmd: ""
          split: horizontal
```

---

## Configuration Precedence

Layouts are loaded in this order (first match wins):

1. `--layout` flag on command line
2. `.peakypanes.yml` in project directory
3. Matching project in `~/.config/peakypanes/config.yml`
4. Built-in `dev-3` layout (default)

---

## Tips

### Keep Crashed Panes Visible

Peaky Panes sets `remain-on-exit: on` by default so panes stay open when commands exit. Set `remain-on-exit: off` in your layout's `tmux_options` if you prefer panes to close normally.

### Prefer `grid:` for Exact Grids

For predictable rows/columns (2x3, 3x4, etc.), use the top-level `grid` configuration. `layout: tiled` is best-effort and may choose a different shape based on window size.

### Empty Commands

Use `cmd: ""` for panes where you want a shell ready for manual commands.

### Log Tailing with Fallbacks

Handle missing log files gracefully:

```yaml
cmd: "tail -F ${log_file} 2>/dev/null || echo 'Waiting for ${log_file}...'"
```

### Filter Noisy Logs

Reduce log noise with grep:

```yaml
cmd: "tail -F ${log_file} | grep -Ev 'DEBUG|TRACE|healthcheck'"
```
