#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
BIN="$BIN_DIR/peky"
RUN_ROOT="$(mktemp -d)"
RUNTIME_DIR="$RUN_ROOT/runtime"
CONFIG_DIR="$RUN_ROOT/config"

mkdir -p "$BIN_DIR"
mkdir -p "$RUNTIME_DIR" "$CONFIG_DIR/layouts"

export PEAKYPANES_RUNTIME_DIR="$RUNTIME_DIR"
export PEAKYPANES_CONFIG_DIR="$CONFIG_DIR"

echo "==> Building peky"
go build -o "$BIN" ./cmd/peky

echo "==> Starting daemon"
"$BIN" daemon restart --yes >/dev/null

cleanup() {
  echo "==> Stopping daemon"
  "$BIN" daemon stop --yes >/dev/null 2>&1 || true
  rm -rf "$RUN_ROOT" >/dev/null 2>&1 || true
}
trap cleanup EXIT

run() {
  echo
  echo "==> $*"
  "$@"
}

run "$BIN" --help
run "$BIN" version
run "$BIN" layouts
run "$BIN" --yes session start --name demo --path . --layout auto
run "$BIN" session list

PANE_ID=""
for i in {1..40}; do
  PANE_ID=$("$BIN" pane list --json 2>/dev/null | python3 -c 'import json,sys; data=json.load(sys.stdin); panes=data.get("data",{}).get("panes",[]) or []; print(panes[0].get("id","") if panes else "")' || true)
  if [[ -n "$PANE_ID" ]]; then
    break
  fi
  sleep 0.25
done

if [[ -z "$PANE_ID" ]]; then
  echo "No panes available to test pane commands" >&2
  exit 1
fi

echo "Using pane: $PANE_ID"

run "$BIN" pane view --pane-id "$PANE_ID" --rows 10 --cols 60 --mode ansi
run "$BIN" --yes pane send --pane-id "$PANE_ID" --text "echo hello"
run "$BIN" --yes pane run --pane-id "$PANE_ID" --command "echo hello"
run "$BIN" pane snapshot --pane-id "$PANE_ID"
run "$BIN" pane history --pane-id "$PANE_ID"
run "$BIN" pane tail --pane-id "$PANE_ID" --lines 5 --follow=false

run "$BIN" --yes pane send --scope session --text "echo broadcast"

run "$BIN" --yes relay create --from "$PANE_ID" --scope session
RELAY_ID=$("$BIN" relay list --json 2>/dev/null | python3 -c 'import json,sys; data=json.load(sys.stdin); relays=data.get("data",{}).get("relays",[]) or []; print(relays[0].get("id","") if relays else "")' || true)
if [[ -n "$RELAY_ID" ]]; then
  run "$BIN" --yes relay stop --id "$RELAY_ID"
fi

run "$BIN" events replay --limit 5

run "$BIN" daemon stop --yes
