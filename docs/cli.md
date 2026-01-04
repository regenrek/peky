# CLI reference

## Basics

```bash
peakypanes                     # Open dashboard (direct)
peakypanes dashboard|ui        # Open dashboard (direct)
peakypanes open|start|o         # Start session and open dashboard
peakypanes <layout>             # Shorthand: start --layout <layout>
peakypanes init                 # Create global config
peakypanes init --local         # Create .peakypanes.yml in cwd
peakypanes layouts              # List available layouts
peakypanes layouts export NAME  # Export layout YAML
peakypanes clone|c USER/REPO    # Clone and start session
peakypanes version              # Show version
peakypanes --version|-v         # Show version
peakypanes help|--help          # Help
```

Global flags (all commands):

```bash
--json        # Emit JSON output (schema: docs/schemas/cli.schema.json)
--timeout     # Override command timeout (Go duration, e.g. 2s, 500ms)
--yes|-y      # Skip confirmations for side-effect commands
--version|-v  # Show version and exit
--fresh-config   # Start with no global config or saved state
--temporary-run  # Use a temporary runtime + config dir (implies --fresh-config)
```

## Command tree

```text
peakypanes
  dashboard|ui
  start|open|o
  daemon [start|stop|restart]
  init
  layouts [export]
  workspace [list|open|close|close-all]
  clone|c
  session [list|start|kill|rename|focus|snapshot]
  pane [list|rename|add|split|close|swap|resize|send|run|view|tail|snapshot|history|wait|tag|action|key|signal|focus]
  relay [create|list|stop|stop-all]
  events [watch|replay]
  context [pack]
  nl [plan|run]
  version
  help|--help
```

## Config, layouts, and clone

```bash
peakypanes init --local --layout auto --force
peakypanes layouts
peakypanes layouts export NAME
peakypanes clone USER/REPO --session NAME --layout LAYOUT --path ./dest
```

## Session + daemon

```bash
peakypanes daemon               # Run daemon in foreground
peakypanes daemon start          # Same as `daemon`
peakypanes daemon stop           # Stop daemon (use --yes to skip confirmation)
peakypanes daemon restart        # Restart daemon (use --yes to skip confirmation)
peakypanes daemon --pprof        # Enable pprof server (requires profiler build)
peakypanes daemon --pprof-addr 127.0.0.1:6060

peakypanes session list
peakypanes session start --name NAME --path PATH --layout LAYOUT --env KEY=VAL
peakypanes session kill --name NAME
peakypanes session rename --old OLD --new NEW
peakypanes session focus --name NAME
peakypanes session snapshot
```

## Workspace

```bash
peakypanes workspace list
peakypanes workspace open --name NAME|--path PATH|--id ID
peakypanes workspace close --name NAME|--path PATH|--id ID
peakypanes workspace close-all
```

## Pane

List and metadata:

```bash
peakypanes pane list [--session NAME]
peakypanes pane view --pane-id PANE --rows 24 --cols 80 --mode ansi|lipgloss|plain
peakypanes pane tail --pane-id PANE [--follow] [--lines 200] [--grep REGEX] [--since RFC3339|DURATION] [--until RFC3339|DURATION]
peakypanes pane snapshot --pane-id PANE [--rows 200]
peakypanes pane history --pane-id PANE [--limit 50] [--since RFC3339]
peakypanes pane wait --pane-id PANE --for REGEX [--timeout 10s]
```

Use `--pane-id @focused` to target the currently focused pane.

Lifecycle and layout:

```bash
peakypanes pane rename --pane-id PANE --name NAME
peakypanes pane rename --session NAME --index INDEX --name NAME
peakypanes pane add [--session NAME] [--index INDEX] [--pane-id PANE] [--orientation vertical|horizontal] [--percent 50] [--focus=false]
peakypanes pane split --session NAME --index INDEX --orientation vertical|horizontal [--percent 50] [--focus=false]
peakypanes pane close --pane-id PANE
peakypanes pane close --session NAME --index INDEX
peakypanes pane swap --session NAME --a INDEX --b INDEX
peakypanes pane resize --pane-id PANE --cols N --rows N
peakypanes pane focus --pane-id PANE
peakypanes pane signal --pane-id PANE --signal TERM
```

By default, `pane add` and `pane split` focus the newly created pane. Use `--focus=false` to keep the current focus.

Input send/run:

```bash
# exactly one of --text/--stdin/--file, and exactly one of --pane-id/--scope
# tool-aware by default (codex/claude/etc) based on pane metadata
peakypanes pane send --pane-id PANE --text "raw bytes"
peakypanes pane send --scope all --text "hello"
peakypanes pane send --pane-id PANE --stdin < payload.txt
peakypanes pane send --pane-id PANE --file ./payload.txt

# run sends payload + submit bytes (tool-aware)
peakypanes pane run --pane-id PANE --command "ls -la"
peakypanes pane run --scope session --command "git status"
peakypanes pane run --pane-id PANE --stdin < cmd.txt
peakypanes pane run --pane-id PANE --file ./cmd.txt

# target only panes running a tool
peakypanes pane run --scope all --command "hello" --tool codex

# bypass tool-aware formatting
peakypanes pane send --pane-id PANE --text "raw bytes" --raw

# delays
peakypanes pane send --pane-id PANE --text "hi" --delay 250ms
peakypanes pane run --pane-id PANE --command "make" --submit-delay 150ms
```

Safety flags for `pane run`:

```bash
peakypanes pane run --pane-id PANE --command "rm -rf /" --confirm
peakypanes pane run --pane-id PANE --command "deploy" --require-ack
```

Tagging:

```bash
peakypanes pane tag add --pane-id PANE --tag build
peakypanes pane tag remove --pane-id PANE --tag build
peakypanes pane tag list --pane-id PANE
```

Scrollback/copy actions and keys:

```bash
# actions: enter_scrollback, exit_scrollback, scroll_up, scroll_down,
# page_up, page_down, scroll_top, scroll_bottom,
# enter_copy, exit_copy, copy_move, copy_page_up, copy_page_down,
# copy_toggle_select, copy_yank
peakypanes pane action --pane-id PANE --action scroll_up --lines 5
peakypanes pane action --pane-id PANE --action copy_move --delta-x 1 --delta-y -1
peakypanes pane key --pane-id PANE --key c --mods ctrl
peakypanes pane key --pane-id PANE --key page_up --scrollback-toggle
```

## Relay

```bash
# exactly one of --to or --scope
peakypanes relay create --from PANE --to PANE --mode line|raw --delay 200ms --prefix "[relay] " --ttl 5m
peakypanes relay create --from PANE --scope session
peakypanes relay list
peakypanes relay stop --id RELAY_ID
peakypanes relay stop-all
```

## Events

```bash
peakypanes events watch [--types pane_updated,toast,focus]
peakypanes events replay [--since 2025-01-01T00:00:00Z] [--until ...] [--limit 100] [--types ...]
```

## Context pack

```bash
peakypanes context pack --include panes,git,errors --max-bytes 200000
```

## NL plan/run

```bash
peakypanes nl plan "list sessions"
peakypanes nl run "start a session named demo in ~/code"
```

## Slash commands (TUI quick reply)

Slash commands are generated from the CLI spec and accept standard CLI flags:

```
/all "message"              # alias: pane send --scope all
/session "message"          # alias: pane send --scope session
/project "message"          # alias: pane send --scope project
```

You can extend or change slash shortcuts in `internal/cli/spec/commands.yaml`.

## JSON output

All `--json` responses follow `docs/schemas/cli.schema.json`. Streaming commands (e.g. `pane tail`, `events watch`) emit frames with `meta.stream=true` and `meta.seq`.
