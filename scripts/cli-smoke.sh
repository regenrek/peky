#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
BIN="$BIN_DIR/peakypanes"

mkdir -p "$BIN_DIR"

echo "==> Building peakypanes"
go build -o "$BIN" ./cmd/peakypanes

echo "==> Starting daemon"
"$BIN" daemon &
DAEMON_PID=$!

cleanup() {
  echo "==> Stopping daemon"
  "$BIN" daemon stop --yes >/dev/null 2>&1 || true
  if kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill -TERM "$DAEMON_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

# wait for daemon to accept commands
for i in {1..40}; do
  if "$BIN" session list >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
  if [[ $i -eq 40 ]]; then
    echo "Daemon did not start in time" >&2
    exit 1
  fi
  done

run() {
  echo
  echo "==> $*"
  "$@"
}

run "$BIN" --help
run "$BIN" version
run "$BIN" layouts
run "$BIN" session start --name demo --path .
run "$BIN" session list

PANE_JSON=$("$BIN" pane list --json)
PANE_ID=$(python3 - <<'PY'
import json,sys
raw=sys.stdin.read()
try:
  data=json.loads(raw)
  panes=data.get('data',{}).get('panes',[])
  print(panes[0]['id'] if panes else '')
except Exception:
  print('')
PY
<<<"$PANE_JSON")

if [[ -z "$PANE_ID" ]]; then
  echo "No panes available to test pane commands" >&2
  exit 1
fi

echo "Using pane: $PANE_ID"

run "$BIN" pane view --pane-id "$PANE_ID" --rows 10 --cols 60 --mode ansi
run "$BIN" pane send --pane-id "$PANE_ID" --text "echo hello"
run "$BIN" pane run --pane-id "$PANE_ID" --command "echo hello"
run "$BIN" pane snapshot --pane-id "$PANE_ID"
run "$BIN" pane history --pane-id "$PANE_ID"
run "$BIN" pane tail --pane-id "$PANE_ID" --lines 5

run "$BIN" pane send --scope session --text "echo broadcast"

run "$BIN" relay create --from "$PANE_ID" --scope session
RELAY_ID=$("$BIN" relay list --json | python3 - <<'PY'
import json,sys
raw=sys.stdin.read()
try:
  data=json.loads(raw)
  relays=data.get('data',{}).get('relays',[])
  print(relays[0]['id'] if relays else '')
except Exception:
  print('')
PY
)
if [[ -n "$RELAY_ID" ]]; then
  run "$BIN" relay stop --id "$RELAY_ID"
fi

run "$BIN" events replay --limit 5

run "$BIN" daemon stop --yes
