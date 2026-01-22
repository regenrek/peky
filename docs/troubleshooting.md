# Troubleshooting

## Running the wrong `peky` (dev build on PATH)

If `peky --version` prints `peky dev`, you're running a locally-built binary (often `~/go/bin/peky`).

Show all `peky` binaries on your PATH:

```bash
type -a peky
```

Ensure you're running the Homebrew-installed release:

```bash
"$(brew --prefix)/bin/peky" --version
```

If `which peky` points to `~/go/bin/peky`, remove it and refresh shell command cache:

```bash
trash "$HOME/go/bin/peky"
hash -r
```

## Fully reset local state (fresh start)

Stop the daemon (kills sessions):

```bash
peky daemon stop --yes
```

Show effective paths:

```bash
peky debug paths
```

Then delete state dirs.

macOS:
```bash
trash "$HOME/Library/Application Support/peky" "$HOME/.config/peky"
```

Linux:
```bash
trash "$HOME/.local/share/peky" "$HOME/.config/peky"
```

## “restored” banner won’t clear

The banner is driven by the restart notice flag stored next to the global config.
Clear it directly:

```bash
trash "$HOME/.config/peky/.pp-restart-notice"
```

## Daemon stuck or restart

The daemon owns sessions and PTYs. If it becomes unresponsive, the only recovery
is a manual restart, which will terminate all running sessions.

Manual restart (macOS default path):

```bash
kill "$(cat "$HOME/Library/Application Support/peky/daemon.pid")"
```

Manual restart (Linux default path):

```bash
kill "$(cat "$HOME/.config/peky/daemon.pid")"
```

If you set custom runtime paths or env overrides, use `peky debug paths` to find the active pid file.
