# Dashboard and keybindings

Running `peky` with no subcommand opens the dashboard UI in the current terminal.

The dashboard shows:
- Projects on top (tabs)
- Sessions on the left (with pane counts and expandable panes)
- Live pane layout canvas on the right (native panes are fully interactive)
- Action line (always visible) and target pane highlight for follow-ups

Action line details:
- Default typing goes to the selected pane (`SOFT`).
- Click the action line to focus it (for `/` slash commands, `@` file picker, and structured actions).
- `enter` submits the action line (pane mode). `esc` clears the action line input.
- Focus shortcut: `ctrl+shift+/` (default; configurable via `dashboard.keymap.focus_action`).

Hard RAW:
- `ctrl+shift+k` toggles `RAW` (pure terminal).
- In `RAW`, the UI does not intercept keys (except `ctrl+shift+k` itself). Mouse resize and selection still work.

## Navigation overview (always visible)

- ctrl+shift+a / ctrl+shift+d project
- ctrl+shift+w / ctrl+shift+s session/panes
- ctrl+shift+up / ctrl+shift+down session only
- ctrl+shift+← / ctrl+shift+→ pane
- ctrl+shift+space last pane
- ctrl+shift+g help

## Key bindings (also shown in the help view)

Keymap overrides are available in the global config (~/.config/peky/config.yml).

Project
- ctrl+shift+o open project picker (creates session detached; stay in dashboard)
- ctrl+shift+b toggle sidebar (show/hide sessions list)
- ctrl+shift+c close project (hides from tabs; sessions keep running; press k in the dialog to kill)

Session
- ctrl+shift+n new session (pick layout)
- ctrl+shift+x close session
- rename session via command palette (ctrl+shift+p)

Window
Pane list
- ctrl+shift+] toggle pane list

Pane
- ctrl+shift+k toggle hard raw (`SOFT`/`RAW`)
- ctrl+shift+r resize mode (keyboard only; arrows resize, tab cycles edges, s toggle snap, 0 reset sizes, z zoom, esc exit; hold option/alt to disable snap)
- mouse: click selects a pane; drag dividers to resize; right-click pane for context menu
- f7 scrollback mode (native only; configurable via dashboard.keymap.scrollback)
- f8 copy mode (native only; configurable via dashboard.keymap.copy_mode)

Mouse + snapping notes
- Drag dividers to resize; corners resize both axes.
- Right-click pane body for context menu.
- Mouse wheel scrolls host scrollback when the pane isn't actively using mouse reporting.
- Hold Shift while scrolling for fine scroll (1 line/step). Hold Ctrl for page scroll.
- Snap is on by default; hold alt/option to disable snap while dragging.
- Ghostty: set right-click to open the terminal context menu so the dashboard can intercept it.
- Keyboard: peky enables the Kitty keyboard protocol (CSI-u) on startup for reliable Ctrl+Shift chords and full key fidelity (Ghostty/kitty/wezterm recommended).

Other
- ctrl+shift+p command palette
- f5 refresh, ctrl+shift+, edit config, ctrl+shift+f filter, ctrl+shift+q quit
- Note: `ctrl+c` is sent to the selected pane (so terminal apps work normally).

## Daemon status (footer)

Bottom-right indicator:
- `up` (dim): daemon reachable
- `restored` (yellow): daemon was restarted; panes may be stale/dead (still viewable)
- `down` (red): daemon unreachable

Click the indicator:
- `restored`: opens a dialog with actions: **Start fresh** or **Check stale panes**
- `down`: prompts to restart the daemon

## Dashboard config (optional)

```yaml
dashboard:
  refresh_ms: 2000
  preview_lines: 12
  preview_compact: true
  idle_seconds: 20
  resize:
    mouse_apply: live  # live | commit
    mouse_throttle_ms: 16
    freeze_content_during_drag: true
  sidebar:
    hidden: false
  attach_behavior: current  # current | detached
  pane_navigation_mode: spatial  # spatial | memory
  quit_behavior: prompt  # prompt | keep | stop
  keymap:
    project_left: ["ctrl+shift+a"]
    project_right: ["ctrl+shift+d"]
    session_up: ["ctrl+shift+w"]
    session_down: ["ctrl+shift+s"]
    session_only_up: ["ctrl+shift+up"]
    session_only_down: ["ctrl+shift+down"]
    pane_next: ["ctrl+shift+right"]
    pane_prev: ["ctrl+shift+left"]
    toggle_last_pane: ["ctrl+shift+space"]
    focus_action: ["ctrl+shift+/"]
    hard_raw: ["ctrl+shift+k"]
    resize_mode: ["ctrl+shift+r"]
    scrollback: ["f7"]
    copy_mode: ["f8"]
    toggle_panes: ["ctrl+shift+]"]
    toggle_sidebar: ["ctrl+shift+b"]
    close_project: ["ctrl+shift+c"]
    command_palette: ["ctrl+shift+p"]
    edit_config: ["ctrl+shift+,"]
    help: ["ctrl+shift+g"]
    quit: ["ctrl+shift+q"]
    filter: ["ctrl+shift+f"]
    open_project: ["ctrl+shift+o"]
    new_session: ["ctrl+shift+n"]
    kill: ["ctrl+shift+x"]
  status_regex:
    success: "(?i)done|finished|success|completed"
    error: "(?i)error|failed|panic"
    running: "(?i)running|in progress|building|installing"
  agent_detection:
    codex: true
    claude: true
```

attach_behavior controls what the attach/start action does (default current):
- current focuses the selected session in the dashboard
- detached creates the session without switching focus

pane_navigation_mode controls left/right navigation across projects and dashboard columns:
- spatial keeps the same row when moving between projects
- memory restores the last selection per project

quit_behavior controls what happens on quit when panes are still running:
- prompt shows a quit dialog (default)
- keep exits immediately and leaves sessions running
- stop stops the daemon (killing all panes)

## Agent status detection (Codex and Claude Code)

peky can read per-pane JSON state files to show accurate running/idle/done status for Codex CLI and Claude Code TUI sessions. This is on by default and falls back to regex or idle detection if no state file is present. You can disable it via dashboard.agent_detection.

State files are written under ${XDG_RUNTIME_DIR:-/tmp}/peky/agent-state and keyed by PEKY_PANE_ID (override with PEKY_AGENT_STATE_DIR).

Codex CLI (TUI)

Add a notify command in your Codex config to call the peky hook script (Codex passes one JSON arg):

```toml
# ~/.codex/config.toml
notify = ["python3", "/absolute/path/to/peakypanes/scripts/agent-state/codex-notify.py"]
```

Tip: with npm i -g peakypanes, the scripts live under $(npm root -g)/peakypanes/scripts/agent-state/.
Note: Codex notify only fires on turn completion, so running state still relies on regex or idle detection between turns.

Claude Code (TUI)

Configure hooks to run the peky hook script (Claude passes JSON on stdin). Recommended events:
SessionStart, UserPromptSubmit, PreToolUse, PermissionRequest, Stop, SessionEnd.

Example hook command (wire it to each event above in Claude Code):

```bash
python3 /absolute/path/to/peakypanes/scripts/agent-state/claude-hook.py
```
