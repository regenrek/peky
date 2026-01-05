#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

gobin="${GOBIN:-}"
if [[ -z "$gobin" ]]; then
  gobin="$(go env GOPATH)/bin"
fi
peky_bin="${gobin}/peky"
peak_bin="${gobin}/peakypanes"

kill_all=false
skip_daemon_restart=false
for arg in "$@"; do
  case "$arg" in
    --all)
      kill_all=true
      ;;
    --no-daemon-restart|--skip-daemon-restart)
      skip_daemon_restart=true
      ;;
  esac
done

echo "reinit: go install ./cmd/peky ./cmd/peakypanes"
go install ./cmd/peky ./cmd/peakypanes

pids="$(pgrep -f "peky.*daemon|peakypanes.*daemon" || true)"
if [[ -z "$pids" ]]; then
  echo "reinit: no running daemon found"
  ps -ax -o pid=,command= | rg "peky" || true
else
  echo "reinit: stopping daemon(s): $pids"
  kill "$pids" || true
fi

if [[ "$kill_all" == true ]]; then
  ui_pids_raw="$(pgrep -x "peky" || true)"
  peak_ui_pids_raw="$(pgrep -x "peakypanes" || true)"
  ui_pids=()
  peak_ui_pids=()
  if [[ -n "$ui_pids_raw" ]]; then
    readarray -t ui_pids <<<"$ui_pids_raw"
  fi
  if [[ -n "$peak_ui_pids_raw" ]]; then
    readarray -t peak_ui_pids <<<"$peak_ui_pids_raw"
  fi
  if (( ${#ui_pids[@]} > 0 || ${#peak_ui_pids[@]} > 0 )); then
    echo "reinit: stopping UI process(es): ${ui_pids[*]} ${peak_ui_pids[*]}"
    kill "${ui_pids[@]}" "${peak_ui_pids[@]}" || true
  fi
fi

sleep 0.5

still_running="$(pgrep -f "peky.*daemon" || true)"
if [[ -n "$still_running" ]]; then
  echo "reinit: force-killing daemon(s): $still_running"
  kill -9 "$still_running" || true
fi

if [[ "$kill_all" == true ]]; then
  still_ui_raw="$(pgrep -x "peky" || true)"
  still_peak_ui_raw="$(pgrep -x "peakypanes" || true)"
  still_ui=()
  still_peak_ui=()
  if [[ -n "$still_ui_raw" ]]; then
    readarray -t still_ui <<<"$still_ui_raw"
  fi
  if [[ -n "$still_peak_ui_raw" ]]; then
    readarray -t still_peak_ui <<<"$still_peak_ui_raw"
  fi
  if (( ${#still_ui[@]} > 0 || ${#still_peak_ui[@]} > 0 )); then
    echo "reinit: force-killing UI process(es): ${still_ui[*]} ${still_peak_ui[*]}"
    kill -9 "${still_ui[@]}" "${still_peak_ui[@]}" || true
  fi
fi

if [[ ! -x "$peky_bin" ]]; then
  echo "reinit: peky binary not found at $peky_bin" >&2
  exit 1
fi
if [[ ! -x "$peak_bin" ]]; then
  echo "reinit: peakypanes binary not found at $peak_bin" >&2
  exit 1
fi

if [[ "$skip_daemon_restart" == true ]]; then
  echo "reinit: skipping daemon restart"
  exit 0
fi

echo "reinit: restarting daemon"
"$peky_bin" daemon restart -y
