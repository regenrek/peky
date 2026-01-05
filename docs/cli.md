# CLI reference

## Basics

The CLI command is `peky` (alias: `peakypanes`).

```bash
peky                     # Open dashboard (direct)
peky dashboard|ui        # Open dashboard (direct)
peky open|start|o         # Start session and open dashboard
peky <layout>             # Shorthand: start --layout <layout>
peky init                 # Create global config
peky init --local         # Create .peakypanes.yml in cwd
peky layouts              # List available layouts
peky layouts export NAME  # Export layout YAML
peky clone|c USER/REPO    # Clone and start session
peky version              # Show version
peky --version|-v         # Show version
peky help|--help          # Help
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
peky
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
peky init --local --layout auto --force
peky layouts
peky layouts export NAME
peky clone USER/REPO --session NAME --layout LAYOUT --path ./dest
```

## Session + daemon

```bash
peky daemon               # Run daemon in foreground
peky daemon start          # Same as `daemon`
peky daemon stop           # Stop daemon (use --yes to skip confirmation)
peky daemon restart        # Restart daemon (use --yes to skip confirmation)
peky daemon --pprof        # Enable pprof server (requires profiler build)
peky daemon --pprof-addr 127.0.0.1:6060

peky session list
peky session start --name NAME --path PATH --layout LAYOUT --env KEY=VAL
peky session kill --name NAME
peky session rename --old OLD --new NEW
peky session focus --name NAME
peky session snapshot
```

## Workspace

```bash
peky workspace list
peky workspace open --name NAME|--path PATH|--id ID
peky workspace close --name NAME|--path PATH|--id ID
peky workspace close-all
```

## Pane

List and metadata:

```bash
peky pane list [--session NAME]
peky pane view --pane-id PANE --rows 24 --cols 80 --mode ansi|plain
peky pane tail --pane-id PANE [--follow] [--lines 200] [--grep REGEX] [--since RFC3339|DURATION] [--until RFC3339|DURATION]
peky pane snapshot --pane-id PANE [--rows 200]
peky pane history --pane-id PANE [--limit 50] [--since RFC3339]
peky pane wait --pane-id PANE --for REGEX [--timeout 10s]
```

Use `--pane-id @focused` to target the currently focused pane.

Lifecycle and layout:

```bash
peky pane rename --pane-id PANE --name NAME
peky pane rename --session NAME --index INDEX --name NAME
peky pane add [--session NAME] [--index INDEX] [--pane-id PANE] [--orientation vertical|horizontal] [--percent 50] [--focus=false]
peky pane split --session NAME --index INDEX --orientation vertical|horizontal [--percent 50] [--focus=false]
peky pane close --pane-id PANE
peky pane close --session NAME --index INDEX
peky pane swap --session NAME --a INDEX --b INDEX
peky pane resize --pane-id PANE --cols N --rows N
peky pane focus --pane-id PANE
peky pane signal --pane-id PANE --signal TERM
```

By default, `pane add` and `pane split` focus the newly created pane. Use `--focus=false` to keep the current focus.

Input send/run:

```bash
# exactly one of --text/--stdin/--file, and exactly one of --pane-id/--scope
# tool-aware by default (codex/claude/etc) based on pane metadata
peky pane send --pane-id PANE --text "raw bytes"
peky pane send --scope all --text "hello"
peky pane send --pane-id PANE --stdin < payload.txt
peky pane send --pane-id PANE --file ./payload.txt

# run sends payload + submit bytes (tool-aware)
peky pane run --pane-id PANE --command "ls -la"
peky pane run --scope session --command "git status"
peky pane run --pane-id PANE --stdin < cmd.txt
peky pane run --pane-id PANE --file ./cmd.txt

# target only panes running a tool
peky pane run --scope all --command "hello" --tool codex

# bypass tool-aware formatting
peky pane send --pane-id PANE --text "raw bytes" --raw

# delays
peky pane send --pane-id PANE --text "hi" --delay 250ms
peky pane run --pane-id PANE --command "make" --submit-delay 150ms
```

Safety flags for `pane run`:

```bash
peky pane run --pane-id PANE --command "rm -rf /" --confirm
peky pane run --pane-id PANE --command "deploy" --require-ack
```

Tagging:

```bash
peky pane tag add --pane-id PANE --tag build
peky pane tag remove --pane-id PANE --tag build
peky pane tag list --pane-id PANE
```

Scrollback/copy actions and keys:

```bash
# actions: enter_scrollback, exit_scrollback, scroll_up, scroll_down,
# page_up, page_down, scroll_top, scroll_bottom,
# enter_copy, exit_copy, copy_move, copy_page_up, copy_page_down,
# copy_toggle_select, copy_yank
peky pane action --pane-id PANE --action scroll_up --lines 5
peky pane action --pane-id PANE --action copy_move --delta-x 1 --delta-y -1
peky pane key --pane-id PANE --key c --mods ctrl
peky pane key --pane-id PANE --key page_up --scrollback-toggle
```

## Relay

```bash
# exactly one of --to or --scope
peky relay create --from PANE --to PANE --mode line|raw --delay 200ms --prefix "[relay] " --ttl 5m
peky relay create --from PANE --scope session
peky relay list
peky relay stop --id RELAY_ID
peky relay stop-all
```

## Events

```bash
peky events watch [--types pane_updated,toast,focus]
peky events replay [--since 2025-01-01T00:00:00Z] [--until ...] [--limit 100] [--types ...]
```

## Context pack

```bash
peky context pack --include panes,git,errors --max-bytes 200000
```

## NL plan/run

```bash
peky nl plan "list sessions"
peky nl run "start a session named demo in ~/code"
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
