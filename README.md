# tmuxman

`tmuxman` is a Charm-powered Go CLI that scaffolds tmux workspaces as deterministic grids. Run it with no flags to open a Bubble Tea form (navbar + pane inputs + shortcut footer), or pass flags for automation-friendly flows. The default layout is a 2x2 grid rooted in your current directory.

## Features
- Interactive terminal form (Bubble Tea + `?` help pane) that collects the session name and grid layout.
- Overlay help: press `?` to toggle a modal explaining navigation keys; `q`/`esc` exit without creating a session.
- Fast flag-driven mode for scripts: `tmuxman -d` uses the defaults immediately, while `--session`, `--layout`, and `-C` let you specify everything up front.
- Resume helper: `tmuxman --resume` lists current tmux sessions so you can switch without remembering names.
- Deterministic pane creation that works for any layout up to 12 panes (e.g. 2x2, 2x3, 3x3). Every pane starts in the same directory so project tooling is ready to go.
- Smart attach behaviour: outside tmux it runs `tmux attach-session`; inside tmux it switches your client, so keyboard focus stays intact.

## Installation
```
go install github.com/kregenrek/tmuxman/cmd/tmuxman@latest
```
Ensure `$GOBIN` (or `$HOME/go/bin`) is on your `PATH` so you can run `tmuxman` from any shell.

## Usage
```
tmuxman                    # interactive prompts for session + layout
tmuxman -d                 # quick defaults: session "grid", layout 2x2
tmuxman --session dev --layout 2x3   # non-interactive custom grid
tmuxman --session web --layout 3x2 -C ~/projects/webapp   # custom start dir
tmuxman -d --no-attach     # provision the session without stealing focus
tmuxman --resume           # pick an existing tmux session to attach to

Inside the interactive form: `tab`/`shift+tab` move between inputs, `enter` submits, `?` toggles help, `q` aborts.
```

### Flags
- `-d`: skip prompts and fall back to `grid` + `2x2` unless overridden by other flags.
- `--session`: set the tmux session name. If the session already exists, tmuxman simply attaches/switches to it.
- `--layout`: grid definition formatted as `<rows>x<columns>` (e.g. `2x3`). Layouts are capped at 12 panes to keep tmux usable.
- `-C`: directory that each pane should start in (defaults to the current working directory).
- `--tmux`: optional path to the tmux binary (falls back to `$PATH`).
- `--timeout`: deadline for tmux commands (default `5s`).
- `--no-attach`: create/configure the session but do not attach or switch to it.
- `--resume`: skip layout creation and interactively choose (or directly specify with `--session`) an existing tmux session to attach.

## Development
```
go test ./...
go run ./cmd/tmuxman --session scratch --layout 2x3 --no-attach
```
The tmux orchestration lives in `internal/tmuxctl`, while layout parsing and validation live in `internal/layout` (with table-driven tests).
