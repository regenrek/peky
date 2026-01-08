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

pid_list() {
  printf "%s" "$1" | tr '\n' ' ' | xargs
}

kill_pids() {
  local label="$1"
  local signal="$2"
  local raw="$3"
  local list
  list="$(pid_list "$raw")"
  if [[ -z "$list" ]]; then
    return 0
  fi
  echo "reinit: ${label}: ${list}"
  while IFS= read -r pid; do
    if [[ -z "$pid" ]]; then
      continue
    fi
    if [[ -n "$signal" ]]; then
      kill "$signal" "$pid" || true
    else
      kill "$pid" || true
    fi
  done <<<"$raw"
}

ui_pids() {
  # ps columns: pid, comm (exe name), args (full argv)
  # Keep daemon alive by never killing processes whose args include " daemon ".
  ps -ax -o pid=,comm=,args= | awk '
    ($2 == "peky" || $2 == "peakypanes") && ($0 !~ /[[:space:]]daemon([[:space:]]|$)/) { print $1 }
  '
}

kill_all=false
skip_daemon_restart=false
keep_daemon=false
for arg in "$@"; do
  case "$arg" in
    --all)
      kill_all=true
      ;;
    --keep-daemon|--no-daemon|--preserve-daemon)
      keep_daemon=true
      skip_daemon_restart=true
      ;;
    --no-daemon-restart|--skip-daemon-restart)
      skip_daemon_restart=true
      ;;
  esac
done

echo "reinit: go install ./cmd/peky ./cmd/peakypanes"
go install ./cmd/peky ./cmd/peakypanes

if [[ "$keep_daemon" == true ]]; then
  echo "reinit: keeping daemon running"
else
  pids="$(pgrep -f "peky.*daemon|peakypanes.*daemon" || true)"
  if [[ -z "$pids" ]]; then
    echo "reinit: no running daemon found"
    ps -ax -o pid=,command= | rg "peky" || true
  else
    kill_pids "stopping daemon(s)" "" "$pids"
  fi
fi

if [[ "$kill_all" == true && "$keep_daemon" != true ]]; then
  ui_pids="$(ui_pids || true)"
  if [[ -n "$(pid_list "$ui_pids")" ]]; then
    kill_pids "stopping UI process(es)" "" "$ui_pids"
  fi
fi

sleep 0.5

if [[ "$keep_daemon" != true ]]; then
  still_running="$(pgrep -f "peky.*daemon" || true)"
  if [[ -n "$still_running" ]]; then
    kill_pids "force-killing daemon(s)" "-9" "$still_running"
  fi
fi

if [[ "$kill_all" == true ]]; then
  still_ui_raw="$(pgrep -x "peky" || true)"
  still_peak_ui_raw="$(pgrep -x "peakypanes" || true)"
  still_ui="$(printf "%s\n%s\n" "$still_ui_raw" "$still_peak_ui_raw")"
  if [[ -n "$(pid_list "$still_ui")" ]]; then
    kill_pids "force-killing UI process(es)" "-9" "$still_ui"
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
