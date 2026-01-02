# Configuration and layouts

See the layout builder guide for detailed layout syntax:
- docs/layout-builder.md

## Project-local configuration (.peakypanes.yml)

Create this file in your project root for team-shared layouts:

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
  # Optional automation inputs:
  # broadcast_send:
  #   - text: "claude"
  #     send_delay_ms: 750
  #     # If send_delay_ms is omitted, waits for first output (default 750ms timeout).
  #     wait_for_output: true
  #     submit: true
  #     submit_delay_ms: 250

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
#
# Grid + panes (per-pane direct_send, overrides commands/titles):
# layout:
#   grid: 2x2
#   panes:
#     - title: pane-1
#       cmd: "claude"
#       direct_send:
#         - text: "give me a bubble sort in typescript and rust and go"
#           send_delay_ms: 750
#           wait_for_output: true
#           submit: true
#           submit_delay_ms: 250

# Optional per-project dashboard overrides
# dashboard:
#   sidebar:
#     hidden: true
```

## Global configuration (~/.config/peakypanes/config.yml)

Use this for personal layouts and multi-project management:

```yaml
# Dashboard UI settings (optional)
# dashboard:
#   project_roots:
#     - ~/projects
#     - ~/code
#
#   performance:
#     preset: max          # low | medium | high | max | custom
#     render_policy: visible # visible | all
#     preview_render:
#       mode: direct       # cached | direct | off
#     # For custom tuning:
#     # preset: custom
#     # pane_views:
#     #   max_concurrency: 6
#     #   max_inflight_batches: 3
#     #   max_batch: 12
#     #   min_interval_focused_ms: 16
#     #   min_interval_selected_ms: 60
#     #   min_interval_background_ms: 150
#     #   timeout_focused_ms: 1000
#     #   timeout_selected_ms: 800
#     #   timeout_background_ms: 600
#     #   pump_base_delay_ms: 0
#     #   pump_max_delay_ms: 25
#     #   force_after_ms: 150
#     #   fallback_min_interval_ms: 100

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

## Variable expansion

Use variables in layouts:

| Variable | Description |
| --- | --- |
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

## Built-in layouts

Core layouts:
- auto (default) - auto-detects .peakypanes.yml or falls back to dev-3
- simple - single pane
- split-v - two vertical panes (left/right)
- split-h - two horizontal panes (top/bottom)
- 2x2 - 4-pane grid
- 3x4 - 12-pane grid
- codex-dev - 2x3 grid (shell + 5 codex)

Additional built-ins:
- dev-2 - editor + shell
- dev-3 - editor + server + shell (default fallback)
- fullstack - editor + server + shell + logs
- go-dev - code/run/test + git
- codex-grid - 2x4 grid running codex in every pane

```bash
# List all layouts
peakypanes layouts

# Export a layout to customize
peakypanes layouts export codex-dev > .peakypanes.yml
```

## Layout detection order

1. --layout flag (highest priority)
2. .peakypanes.yml in current directory
3. Project entry in ~/.config/peakypanes/config.yml
4. Built-in dev-3 layout (fallback)
