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

Then delete state dirs.

macOS:
```bash
trash "$HOME/Library/Application Support/peky" "$HOME/.config/peky"
trash "$HOME/Library/Application Support/peakypanes" "$HOME/.config/peakypanes"
```

Linux:
```bash
trash "$HOME/.local/share/peky" "$HOME/.config/peky"
trash "$HOME/.local/share/peakypanes" "$HOME/.config/peakypanes"
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

You can also set PEKY_DAEMON_PID to control the pid file location.
