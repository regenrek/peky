#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

./scripts/reinit.sh --all

daemon_log="/tmp/peakypanes-daemon.out"
ui_log="/tmp/peakypanes-debug.log"

: > "$daemon_log"
: > "$ui_log"

peky daemon >"$daemon_log" 2>&1 &

export PEAKYPANES_DEBUG_EVENTS=1
export PEAKYPANES_DEBUG_EVENTS_LOG="$ui_log"
export PEAKYPANES_PERF_DEBUG=1

# Keep TUI on stdout, redirect stderr logs to file so the UI isn't polluted.
peky start 2>>"$ui_log"
