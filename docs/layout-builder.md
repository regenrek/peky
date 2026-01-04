# Layout Builder Guide

This guide covers everything you need to know about creating custom layouts in Peaky Panes, including pane arrangements and advanced configuration.

## Table of Contents

- [Basic Structure](#basic-structure)
- [Pane Layouts](#pane-layouts)
- [Split Directions](#split-directions)
- [Variables](#variables)
- [Examples](#examples)
- [Configuration Precedence](#configuration-precedence)
- [Tips](#tips)

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
```

---

## Pane Layouts

### Split Layouts (Recommended)

Use `split` on panes after the first to control how the layout is divided. The first pane defines the base area; each subsequent pane splits the remaining space of that base pane, so order matters.

```yaml
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
```

### Exact Grid Example (2x2)

Use `grid` when you need predictable rows/columns:

```yaml
grid: 2x2
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

You can also combine `grid` with `panes` to override per-pane settings (including automation sends):

```yaml
grid: 2x2
panes:
  - title: codex
    cmd: "codex"
    direct_send:
      - text: "hello from pane 1"
        send_delay_ms: 500
  - title: dev
    cmd: "npm run dev"
  - title: logs
    cmd: "tail -f app.log"
  - title: shell
    cmd: ""
```

---

## Split Directions

For precise control, use `split` on individual panes:

| Split | Result |
|-------|--------|
| `horizontal` | Creates left/right panes |
| `vertical` | Creates top/bottom panes |

```yaml
panes:
  - title: editor        # First pane (no split)
    cmd: "${EDITOR:-}"
    size: "60%"
  - title: server        # Splits horizontally from editor
    cmd: "npm run dev"
    split: horizontal
  - title: shell         # Splits vertically from editor's remaining space
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

  panes:
    - title: rust
      cmd: "tail -F ${rust_log}"
    - title: codex
      cmd: "tail -F ${codex_log}"
```

---

## Large Layouts

When you need many panes, prefer a `grid:` layout for predictable sizing.

---

## Automation Sends

Use `broadcast_send` to send input to every pane after it starts, or `direct_send` to send input to a specific pane. Each action can specify a `send_delay_ms` (default: 750ms). If `send_delay_ms` is omitted, the send waits for the pane’s first output (up to the default delay). A trailing newline is added automatically unless you set `submit: true`, which sends Enter separately (optionally delayed via `submit_delay_ms`). When `wait_for_output: true`, the send waits for the pane’s first output and uses `send_delay_ms` as a fallback timeout.

```yaml
layout:
  command: "claude"
  broadcast_send:
    - text: "give me a bubble sort in typescript and rust and go"
      send_delay_ms: 750
      wait_for_output: true
      submit: true
      submit_delay_ms: 250

  panes:
    - title: pane-1
      cmd: "claude"
      direct_send:
        - text: "pane-specific follow-up"
          send_delay_ms: 1000
          wait_for_output: true
```

---

## Examples

### Full-Stack Web Development

```yaml
session: webapp

layout:
  name: fullstack
  description: "Full-stack development with logs"

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
    - title: frontend
      cmd: "tail -f logs/frontend.log"
      split: vertical
    - title: backend
      cmd: "tail -f logs/backend.log"
      split: vertical
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

  panes:
    - title: codex
      cmd: "RUST_LOG=debug codex"
      size: "50%"
    - title: bun-dev
      cmd: "bun dev:tauri"
      split: horizontal
    - title: codex-logs
      cmd: "tail -F ${codex_log} | grep -Ev '\\bINFO\\b'"
      split: vertical
    - title: rust-logs
      cmd: "tail -F ${rust_log}"
      split: vertical
```

### Go Development

```yaml
session: go-project

layout:
  name: go-dev

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
    - title: lazygit
      cmd: "lazygit"
```

### Simple 2-Pane Layout

```yaml
layout:
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
4. Built-in `auto` layout (default)

---

## Tips

### Prefer `grid:` for Exact Grids

For predictable rows/columns (2x3, 3x4, etc.), use the top-level `grid` configuration.

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
