# Custom Layout Support for Custom Panes

## Summary

Added support for per-project `.peakypanes.yml` config files with flexible grid layouts and per-pane startup commands.

## Features Implemented

### 1. New `peakypanes` CLI Command

**File:** `cmd/peakypanes/main.go`

- Auto-detects `.peakypanes.yml` in current directory
- `peakypanes start` — launches tmux session directly
- `peakypanes start --config custom.yml` — uses custom config file
- `peakypanes` — opens interactive TUI

### 2. Per-Project Config Format

**File:** `.peakypanes.yml` (in any project directory)

```yaml
name: myproject
session: myproject
layout: 2x4
panes:
  - cmd: npm run dev
  - cmd: npm run watch
  - cmd: nvim .
  - cmd: lazygit
  - cmd: ""
  - cmd: ""
  - cmd: ""
  - cmd: tail -f logs/app.log
```

### 3. Dynamic Grid Layouts

**File:** `internal/tui/peakypanes/model.go`

- Any `NxM` grid now works (e.g., `2x4`, `4x2`, `3x3`)
- No longer limited to hardcoded presets
- Added grid presets to TUI layout picker: `2x2`, `2x3`, `2x4`, `3x2`, `3x3`

### 4. Per-Pane Commands

**File:** `internal/tmuxctl/client.go`

- Added `PaneCommands []string` to `Options` struct
- Added `SendKeys()` method to send commands to specific panes
- Modified `createGrid()` to track pane IDs and send commands after creation
- Panes filled left-to-right, top-to-bottom

### 5. Single Command for All Panes

```yaml
command: htop  # runs in all 8 panes
```

## Files Changed

| File | Changes |
|------|---------|
| `cmd/peakypanes/main.go` | **NEW** — CLI for per-project configs |
| `internal/tmuxctl/client.go` | Added `SendKeys()`, `PaneCommands`, updated `createGrid()` |
| `internal/tui/peakypanes/model.go` | Added grid presets, dynamic layout support |
| `README.md` | Rewritten for Peaky Panes |
| `CHANGELOG.md` | **NEW** — Version history |

## Usage

```bash
# Install
go install ./cmd/peakypanes

# Create config in project
cat > .peakypanes.yml << 'EOF'
name: myproject
session: myproject
layout: 2x4
panes:
  - cmd: npm run dev
  - cmd: npm run watch
  - cmd: nvim .
  - cmd: lazygit
  - cmd: ""
  - cmd: ""
  - cmd: ""
  - cmd: tail -f logs/app.log
EOF

# Start
peakypanes start
```

## Layout Reference

```
2x4 layout (8 panes):
┌───────┬───────┬───────┬───────┐
│ pane1 │ pane2 │ pane3 │ pane4 │
├───────┼───────┼───────┼───────┤
│ pane5 │ pane6 │ pane7 │ pane8 │
└───────┴───────┴───────┴───────┘

Pane order: left-to-right, top-to-bottom
```

