# Daemon permissions and user impact

peky runs a per-user daemon that owns sessions and PTYs. It is local-only and does not require admin/root permissions.

## User impact

- No special setup is required.
- The daemon starts automatically when needed and runs under the current user.
- There are no network listeners; the IPC endpoint is local to the user.

## Files and permissions

The daemon stores its socket, pid file, and logs under a per-user runtime directory.

- Directory permissions: `0700`
- Socket permissions: `0600` (or `0700` where applicable)
- PID/log files: `0600`

This ensures only the current user can connect.

## Platform notes

macOS / Linux

- Use a per-user runtime path (for example `~/.config/peky/sessiond/`).
- Ensure `mkdir` uses `0700` and apply restrictive perms to the socket.

Windows

- Use a per-user named pipe (e.g. `\\.\pipe\peky-<uid>`).
- No admin rights should be required; rely on default ACLs for the current user.

## Optional system services

If users prefer a background service (LaunchAgent/systemd), it should run as the user and only read/write within that user’s home directory. No elevated permissions are required.
