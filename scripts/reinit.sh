#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

kill_all=false
for arg in "$@"; do
  case "$arg" in
    --all)
      kill_all=true
      ;;
  esac
done

echo "reinit: go install ./cmd/peakypanes"
go install ./cmd/peakypanes

pids="$(pgrep -f "peakypanes.*daemon" || true)"
if [[ -z "$pids" ]]; then
  echo "reinit: no running daemon found"
  ps -ax -o pid=,command= | rg "peakypanes" || true
  if [[ "$kill_all" != true ]]; then
    exit 0
  fi
else
  echo "reinit: stopping daemon(s): $pids"
  kill $pids || true
fi

if [[ "$kill_all" == true ]]; then
  ui_pids="$(pgrep -x "peakypanes" || true)"
  if [[ -n "$ui_pids" ]]; then
    echo "reinit: stopping UI process(es): $ui_pids"
    kill $ui_pids || true
  fi
fi

sleep 0.5

still_running="$(pgrep -f "peakypanes.*daemon" || true)"
if [[ -n "$still_running" ]]; then
  echo "reinit: force-killing daemon(s): $still_running"
  kill -9 $still_running || true
fi

if [[ "$kill_all" == true ]]; then
  still_ui="$(pgrep -x "peakypanes" || true)"
  if [[ -n "$still_ui" ]]; then
    echo "reinit: force-killing UI process(es): $still_ui"
    kill -9 $still_ui || true
  fi
fi
