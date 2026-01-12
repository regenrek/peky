# Troubleshooting

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
