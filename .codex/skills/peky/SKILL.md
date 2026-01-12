---
name: peky
description: Use when operating peky from the CLI or TUI, especially for AI agents who need reliable, low-error procedures. Covers how to target sessions/panes correctly, use scopes, avoid confirmation prompts, and keep CLI/TUI/daemon in sync.
---

# peky

## Overview
Use this skill to operate peky safely and predictably from the CLI, especially when automating or controlling panes via an AI agent.

## Quick Rules (read first)
- **Do not guess flags.** Use `peky <command> --help` before running.
- **Use `--yes` for side effects** to avoid hanging prompts in non-interactive runs.
- **Scopes are only `session|project|all`.** Never pass project names to `--scope`.
- **Project scope requires focus.** If you see "focused project unavailable," run `peky session focus --name <session>` first.
- **Prefer pane IDs for precision.** Use `pane list --json` and pick the pane `id`.
- **Shortcut:** use `--pane-id @focused` to target the currently focused pane.
- **`pane add` defaults to active pane.** For deterministic automation, pass `--pane-id` or `--session` + `--index`.

## Targeting: Session vs Project vs Pane ID
1) **Find session names**  
```bash
peky session list
```

2) **Focus the session you intend to operate on**  
```bash
peky session focus --name "<session>" --yes
```

3) **Send to a specific pane (recommended)**  
```bash
peky pane list --session "<session>" --json
peky pane send --pane-id "<pane-id>" --text "hello world" --yes
```

4) **Send to the whole session or project**
```bash
peky pane send --scope session --text "hello session" --yes
peky pane send --scope project --text "hello project" --yes
peky pane send --scope all --text "hello all" --yes
```

If `--scope project` fails, the focused session is missing or has no project path. **Fix by focusing a session with a valid path.**

## Run vs Send
Use this to avoid confusion in Codex/Claude sessions:
- **`pane run` = execute now.** Sends the command and a newline. Best for shell commands or anything you want to run immediately.
- **`pane send` = type only.** Sends raw input without pressing Enter. Best when you need to compose input step‑by‑step or control submission.
- **Submitting after `send`:** use `pane key` to press Enter when you decide to submit.
- **Tool-aware input (default):** send/run uses pane tool metadata (Codex/Claude/etc) to format input (bracketed paste + submit). Use `--raw` to bypass. Use `--tool` to target only panes running a specific tool.

Examples:
```bash
# Run a shell command (executes immediately)
peky pane run --pane-id "<pane-id>" --command "ls -la" --yes

# Run on the currently focused pane (no lookup needed)
peky pane run --pane-id @focused --command "ls -la" --yes

# Type into a prompt without submitting yet
peky pane send --pane-id "<pane-id>" --text "draft message" --yes
peky pane key --pane-id "<pane-id>" --key enter --yes

# Target only Codex panes in a broadcast
peky pane run --scope all --command "hello" --tool codex --yes

# Send raw bytes (no tool formatting)
peky pane send --pane-id "<pane-id>" --text "raw bytes" --raw --yes
```

## Adding Panes
`pane add` is the first-class command. It defaults to the focused session + active pane.
```bash
peky pane add --yes
peky pane add --session "<session>" --index 3 --orientation horizontal --yes
peky pane add --pane-id "<pane-id>" --yes
```
Use `pane split` when you want an explicit split target and no defaults.

## When the TUI is running
The CLI talks to the same daemon. You can run CLI commands from another terminal:
- If the TUI was launched with `--temporary-run`, the CLI must share that runtime/config env. Otherwise it won't connect.

## Fresh / Temporary Runs (safe testing)
Use these when you don't want to touch existing state:
```bash
peky --fresh-config ...
peky --temporary-run ...
```

## Daemon Management
```bash
peky daemon            # foreground
peky daemon stop --yes
peky daemon restart --yes
peky daemon --pprof    # requires profiler build
```

## JSON + Automation
Use `--json` for machine output and `--timeout` to avoid hangs:
```bash
peky pane list --json
peky events watch --json
peky pane tail --json --lines 50
```

## Stress + E2E (local + self-hosted)
Use the built-in stress battery for regression protection:
```bash
scripts/cli-stress.sh
```

Enable tool loops during stress (Codex/Claude):
```bash
RUN_TOOLS=1 scripts/cli-stress.sh
```

Tip: when using a local binary alias, always call `$PP`, not `PP`:
```bash
PP=./bin/peky
$PP version
```

## Logs (macOS default)
If you need daemon logs:
```bash
tail -n 200 "$HOME/Library/Application Support/peky/daemon.log"
```

## Troubleshooting Checklist (fast)
- **"focused project unavailable"** -> run `session focus --name <session>`
- **Prompt hangs** -> add `--yes`
- **Wrong pane** -> use `pane list --json`, target by `id`
- **No daemon** -> `peky daemon` or `peky daemon start`
- **Can't connect** -> ensure same runtime env or stop old daemon
