# Dashboard and keybindings

Running `peky` with no subcommand opens the dashboard UI in the current terminal.

The dashboard shows:
- Projects on top (tabs)
- Sessions on the left (with pane counts and expandable panes)
- Live pane preview on the right (native panes are fully interactive)
- Input bar (always visible) and target pane highlight for follow-ups

Quick reply details:
- The input is always active; type and press enter to send to the highlighted pane.
- Use esc to clear.
- Type / to see slash commands and press tab to autocomplete.
- Toggle terminal focus to send raw keystrokes into the pane.

## Navigation overview (always visible)

- ctrl+q/ctrl+e project
- ctrl+w/ctrl+s session/panes
- alt+w/alt+s session only
- ctrl+a/ctrl+d pane
- ctrl+g help

## Key bindings (also shown in the help view)

Keymap overrides are available in the global config (~/.config/peakypanes/config.yml).

Project
- ctrl+o open project picker (creates session detached; stay in dashboard)
- ctrl+b toggle sidebar (show/hide sessions list)
- alt+c close project (hides from tabs; sessions keep running; press k in the dialog to kill)

Session
- enter attach/start session (when reply is empty)
- ctrl+n new session (pick layout)
- ctrl+x kill session
- rename session via command palette (ctrl+p)

Window
Pane list
- ctrl+u toggle pane list

Pane
- rename pane via command palette (ctrl+p)
- ctrl+y peek selected pane in new terminal
- ctrl+k toggle terminal focus (native only; configurable via dashboard.keymap.terminal_focus)
- mouse: single-click selects a pane; double-click toggles terminal focus (native only); esc exits focus
- f7 scrollback mode (native only; configurable via dashboard.keymap.scrollback)
- f8 copy mode (native only; configurable via dashboard.keymap.copy_mode)

Other
- ctrl+p command palette
- ctrl+r refresh, ctrl+, edit config, ctrl+f filter, ctrl+c quit

## Dashboard config (optional)

```yaml
dashboard:
  refresh_ms: 2000
  preview_lines: 12
  preview_compact: true
  idle_seconds: 20
  preview_mode: grid  # grid | layout
  sidebar:
    hidden: false
  attach_behavior: current  # current | detached
  pane_navigation_mode: spatial  # spatial | memory
  quit_behavior: prompt  # prompt | keep | stop
  keymap:
    project_left: ["ctrl+q"]
    project_right: ["ctrl+e"]
    session_up: ["ctrl+w"]
    session_down: ["ctrl+s"]
    session_only_up: ["alt+w"]
    session_only_down: ["alt+s"]
    pane_next: ["ctrl+d"]
    pane_prev: ["ctrl+a"]
    terminal_focus: ["ctrl+k"]
    scrollback: ["f7"]
    copy_mode: ["f8"]
    toggle_panes: ["ctrl+u"]
    toggle_sidebar: ["ctrl+b"]
    close_project: ["alt+c"]
    command_palette: ["ctrl+p"]
    edit_config: ["ctrl+," ]
    help: ["ctrl+g"]
    quit: ["ctrl+c"]
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

PeakyPanes can read per-pane JSON state files to show accurate running/idle/done status for Codex CLI and Claude Code TUI sessions. This is on by default and falls back to regex or idle detection if no state file is present. You can disable it via dashboard.agent_detection.

State files are written under ${XDG_RUNTIME_DIR:-/tmp}/peakypanes/agent-state and keyed by PEAKYPANES_PANE_ID (override with PEAKYPANES_AGENT_STATE_DIR).

Codex CLI (TUI)

Add a notify command in your Codex config to call the PeakyPanes hook script (Codex passes one JSON arg):

```toml
# ~/.codex/config.toml
notify = ["python3", "/absolute/path/to/peakypanes/scripts/agent-state/codex-notify.py"]
```

Tip: with npm i -g peakypanes, the scripts live under $(npm root -g)/peakypanes/scripts/agent-state/.
Note: Codex notify only fires on turn completion, so running state still relies on regex or idle detection between turns.

Claude Code (TUI)

Configure hooks to run the PeakyPanes hook script (Claude passes JSON on stdin). Recommended events:
SessionStart, UserPromptSubmit, PreToolUse, PermissionRequest, Stop, SessionEnd.

Example hook command (wire it to each event above in Claude Code):

```bash
python3 /absolute/path/to/peakypanes/scripts/agent-state/claude-hook.py
```
