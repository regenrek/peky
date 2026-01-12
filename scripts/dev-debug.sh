#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

./scripts/reinit.sh --all

daemon_log="/tmp/peky-daemon.out"
ui_log="/tmp/peky-debug.log"

: > "$daemon_log"
: > "$ui_log"

peky daemon >"$daemon_log" 2>&1 &

export PEKY_DEBUG_EVENTS=1
export PEKY_DEBUG_EVENTS_LOG="$ui_log"
export PEKY_PERF_DEBUG=1

# Keep TUI on stdout, redirect stderr logs to file so the UI isn't polluted.
peky start 2>>"$ui_log"
