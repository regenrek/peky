#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

gobin="${GOBIN:-}"
if [[ -z "$gobin" ]]; then
  gobin="$(go env GOPATH)/bin"
fi
peky_bin="${gobin}/peky"

kill_all=false
for arg in "$@"; do
  case "$arg" in
    --all)
      kill_all=true
      ;;
  esac
done

echo "reinit: go install ./cmd/peky"
go install ./cmd/peky

pids="$(pgrep -f "peky.*daemon" || true)"
if [[ -z "$pids" ]]; then
  echo "reinit: no running daemon found"
  ps -ax -o pid=,command= | rg "peky" || true
else
  echo "reinit: stopping daemon(s): $pids"
  kill "$pids" || true
fi

if [[ "$kill_all" == true ]]; then
  ui_pids="$(pgrep -x "peky" || true)"
  if [[ -n "$ui_pids" ]]; then
    echo "reinit: stopping UI process(es): $ui_pids"
    kill "$ui_pids" || true
  fi
fi

sleep 0.5

still_running="$(pgrep -f "peky.*daemon" || true)"
if [[ -n "$still_running" ]]; then
  echo "reinit: force-killing daemon(s): $still_running"
  kill -9 "$still_running" || true
fi

if [[ "$kill_all" == true ]]; then
  still_ui="$(pgrep -x "peky" || true)"
  if [[ -n "$still_ui" ]]; then
    echo "reinit: force-killing UI process(es): $still_ui"
    kill -9 "$still_ui" || true
  fi
fi

if [[ ! -x "$peky_bin" ]]; then
  echo "reinit: peky binary not found at $peky_bin" >&2
  exit 1
fi

echo "reinit: restarting daemon"
"$peky_bin" daemon restart -y
