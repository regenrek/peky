---
name: peky
description: Use when operating PeakyPanes from the CLI or TUI, especially for AI agents who need reliable, low-error procedures. Covers how to target sessions/panes correctly, use scopes, avoid confirmation prompts, and keep CLI/TUI/daemon in sync.
---

# Peky

## Overview
Use this skill to operate PeakyPanes safely and predictably from the CLI, especially when automating or controlling panes via an AI agent.

## Quick Rules (read first)
- **Do not guess flags.** Use `peakypanes <command> --help` before running.
- **Use `--yes` for side effects** to avoid hanging prompts in non-interactive runs.
- **Scopes are only `session|project|all`.** Never pass project names to `--scope`.
- **Project scope requires focus.** If you see "focused project unavailable," run `peakypanes session focus --name <session>` first.
- **Prefer pane IDs for precision.** Use `pane list --json` and pick the pane `id`.
- **`pane add` defaults to active pane.** For deterministic automation, pass `--pane-id` or `--session` + `--index`.

## Targeting: Session vs Project vs Pane ID
1) **Find session names**  
```bash
peakypanes session list
```

2) **Focus the session you intend to operate on**  
```bash
peakypanes session focus --name "<session>" --yes
```

3) **Send to a specific pane (recommended)**  
```bash
peakypanes pane list --session "<session>" --json
peakypanes pane send --pane-id "<pane-id>" --text "hello world" --yes
```

4) **Send to the whole session or project**
```bash
peakypanes pane send --scope session --text "hello session" --yes
peakypanes pane send --scope project --text "hello project" --yes
peakypanes pane send --scope all --text "hello all" --yes
```

If `--scope project` fails, the focused session is missing or has no project path. **Fix by focusing a session with a valid path.**

## Run vs Send
- `pane send` sends raw input (no newline)
- `pane run` sends input + newline  
```bash
peakypanes pane run --pane-id "<pane-id>" --command "ls -la" --yes
```

## Adding Panes
`pane add` is the first-class command. It defaults to the focused session + active pane.
```bash
peakypanes pane add --yes
peakypanes pane add --session "<session>" --index 3 --orientation horizontal --yes
peakypanes pane add --pane-id "<pane-id>" --yes
```
Use `pane split` when you want an explicit split target and no defaults.

## When the TUI is running
The CLI talks to the same daemon. You can run CLI commands from another terminal:
- If the TUI was launched with `--temporary-run`, the CLI must share that runtime/config env. Otherwise it won't connect.

## Fresh / Temporary Runs (safe testing)
Use these when you don't want to touch existing state:
```bash
peakypanes --fresh-config ...
peakypanes --temporary-run ...
```

## Daemon Management
```bash
peakypanes daemon            # foreground
peakypanes daemon stop --yes
peakypanes daemon restart --yes
peakypanes daemon --pprof    # requires profiler build
```

## JSON + Automation
Use `--json` for machine output and `--timeout` to avoid hangs:
```bash
peakypanes pane list --json
peakypanes events watch --json
peakypanes pane tail --json --lines 50
```

## Troubleshooting Checklist (fast)
- **"focused project unavailable"** -> run `session focus --name <session>`
- **Prompt hangs** -> add `--yes`
- **Wrong pane** -> use `pane list --json`, target by `id`
- **No daemon** -> `peakypanes daemon` or `peakypanes daemon start`
- **Can't connect** -> ensure same runtime env or stop old daemon
