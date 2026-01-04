#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
BIN="$BIN_DIR/peakypanes"
SESSION_NAME="stress-$(date +%s)"
RUN_TOOLS="${RUN_TOOLS:-0}"
PANE_COUNT="${PANE_COUNT:-200}"
VIEW_STORM_PANES="${VIEW_STORM_PANES:-50}"
VIEW_STORM_PASSES="${VIEW_STORM_PASSES:-10}"
VIEW_STORM_CONCURRENCY="${VIEW_STORM_CONCURRENCY:-8}"
OUTPUT_FLOOD_LINES="${OUTPUT_FLOOD_LINES:-200000}"

is_uint() {
  [[ "${1:-}" =~ ^[0-9]+$ ]]
}

clamp_uint() {
  local v="$1"
  local min="$2"
  local max="$3"
  if [[ "$v" -lt "$min" ]]; then
    echo "$min"
    return
  fi
  if [[ "$v" -gt "$max" ]]; then
    echo "$max"
    return
  fi
  echo "$v"
}

validate_and_clamp_uint() {
  local name="$1"
  local v="$2"
  local min="$3"
  local max="$4"
  if ! is_uint "$v"; then
    echo "$name must be a non-negative integer (got: $v)" >&2
    exit 1
  fi
  local clamped
  clamped=$(clamp_uint "$v" "$min" "$max")
  if [[ "$clamped" != "$v" ]]; then
    echo "WARN: clamping $name=$v to $clamped (allowed range: $min..$max)" >&2
  fi
  echo "$clamped"
}

PANE_COUNT=$(validate_and_clamp_uint "PANE_COUNT" "$PANE_COUNT" 1 1000)
VIEW_STORM_PANES=$(validate_and_clamp_uint "VIEW_STORM_PANES" "$VIEW_STORM_PANES" 1 500)
VIEW_STORM_PASSES=$(validate_and_clamp_uint "VIEW_STORM_PASSES" "$VIEW_STORM_PASSES" 1 50)
VIEW_STORM_CONCURRENCY=$(validate_and_clamp_uint "VIEW_STORM_CONCURRENCY" "$VIEW_STORM_CONCURRENCY" 1 32)
OUTPUT_FLOOD_LINES=$(validate_and_clamp_uint "OUTPUT_FLOOD_LINES" "$OUTPUT_FLOOD_LINES" 1 2000000)

mkdir -p "$BIN_DIR"

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  echo "==> Building peakypanes"
  go build -o "$BIN" ./cmd/peakypanes
fi

RUNTIME_DIR="$(mktemp -d)"
CONFIG_DIR="$(mktemp -d)"
chmod 700 "$RUNTIME_DIR" "$CONFIG_DIR" >/dev/null 2>&1 || true
export PEAKYPANES_RUNTIME_DIR="$RUNTIME_DIR"
export PEAKYPANES_CONFIG_DIR="$CONFIG_DIR"
export PEAKYPANES_FRESH_CONFIG=1

"$BIN" daemon &
DAEMON_PID=$!

cleanup() {
  "$BIN" session kill --name "$SESSION_NAME" --yes >/dev/null 2>&1 || true
  "$BIN" daemon stop --yes >/dev/null 2>&1 || true
  if kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill -TERM "$DAEMON_PID" >/dev/null 2>&1 || true
  fi
  rm -rf "$RUNTIME_DIR" "$CONFIG_DIR" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# Wait for daemon to accept commands
for i in {1..50}; do
  if "$BIN" session list >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
  if [[ $i -eq 50 ]]; then
    echo "Daemon did not start in time" >&2
    exit 1
  fi
done

"$BIN" session start --name "$SESSION_NAME" --path "$ROOT_DIR" --yes
"$BIN" session focus --name "$SESSION_NAME" --yes
"$BIN" pane add --session "$SESSION_NAME" --yes

PANE0_ID=$("$BIN" pane list --session "$SESSION_NAME" --json 2>/dev/null | jq -r '.data.panes[0].id' 2>/dev/null || true)
if [[ -z "${PANE0_ID}" ]]; then
  PANE0_ID=$("$BIN" pane list --session "$SESSION_NAME" --json 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["panes"][0]["id"])' 2>/dev/null || true)
fi
if [[ -z "${PANE0_ID}" ]]; then
  echo "Failed to resolve initial pane id" >&2
  exit 1
fi

PAYLOAD_FILE="${TMPDIR:-/tmp}/pp-block.txt"
export PAYLOAD_FILE
if command -v python3 >/dev/null 2>&1; then
  python3 - <<'PY'
import os
path = os.environ["PAYLOAD_FILE"]
with open(path, "w") as f:
    f.write("A" * (1024 * 1024))
print(path)
PY
elif command -v python >/dev/null 2>&1; then
  python - <<'PY'
import os
path = os.environ["PAYLOAD_FILE"]
with open(path, "w") as f:
    f.write("A" * (1024 * 1024))
print(path)
PY
else
  head -c 1048576 /dev/zero | tr '\0' 'A' > "$PAYLOAD_FILE"
  echo "$PAYLOAD_FILE"
fi

fail_total=0

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Missing required command: $cmd" >&2
    exit 1
  fi
}

scale_to_panes() {
  if [[ "$PANE_COUNT" -le 1 ]]; then
    return
  fi
  echo "=== 0. Pane scale-up ($PANE_COUNT panes) ==="
  local to_add=$((PANE_COUNT - 1))
  if [[ "$to_add" -le 0 ]]; then
    return
  fi
  seq 1 "$to_add" | xargs -n1 -P4 -I{} "$BIN" pane add --session "$SESSION_NAME" --yes >/dev/null
  local got
  got=$("$BIN" pane list --session "$SESSION_NAME" --json | jq '.data.panes | length')
  echo "panes=$got"
  if [[ "$got" -lt "$PANE_COUNT" ]]; then
    echo "Expected at least $PANE_COUNT panes, got $got" >&2
    fail_total=1
  fi
}

run_snapshot_storm() {
  echo "=== 1. Sequential snapshot storm (200) ==="
  local fail=0
  for i in {1..200}; do
    if ! "$BIN" pane list --session "$SESSION_NAME" --json >/dev/null 2>&1; then
      echo "FAIL[$i]"
      fail=$((fail+1))
    fi
  done
  echo "snapshot-fail=$fail"
  if [[ $fail -ne 0 ]]; then
    fail_total=1
  fi
}

run_parallel_snapshot() {
  echo "=== 2. Parallel snapshot storm (8x50) ==="
  local tmp
  tmp=$(mktemp)
  for _ in {1..50}; do
    (
      if "$BIN" pane list --session "$SESSION_NAME" --json >/dev/null 2>&1; then
        echo ok >>"$tmp"
      else
        echo fail >>"$tmp"
      fi
    ) &
  done
  wait
  local ok_count
  local fail_count
  ok_count=$(grep -c ok "$tmp" 2>/dev/null || true)
  fail_count=$(grep -c fail "$tmp" 2>/dev/null || true)
  rm -f "$tmp"
  echo "parallel-ok=$ok_count fail=$fail_count"
  if [[ $fail_count -ne 0 ]]; then
    fail_total=1
  fi
}

run_osc_flood() {
  echo "=== 3. OSC color query flood (200) ==="
  local fail=0
  for i in {1..200}; do
    out=$("$BIN" pane run --scope session --command "printf \"\\033]10;?\\a\\033]11;?\\a\\033]12;?\\a\"" --yes 2>&1) || {
      echo "FAIL[$i]: $out"
      fail=$((fail+1))
    }
  done
  echo "osc-fail=$fail"
  if [[ $fail -ne 0 ]]; then
    fail_total=1
  fi
}

run_payload_send() {
  echo "=== 4. Large payload send (20x1MB) ==="
  local fail=0
  for i in {1..20}; do
    if ! "$BIN" pane send --scope session --file "$PAYLOAD_FILE" --yes >/dev/null 2>&1; then
      echo "FAIL[$i]"
      fail=$((fail+1))
    fi
  done
  echo "payload-fail=$fail"
  if [[ $fail -ne 0 ]]; then
    fail_total=1
  fi
}

run_mixed_fanout() {
  echo "=== 5. Mixed fan-out + snapshot (100 jobs) ==="
  local tmp
  tmp=$(mktemp)
  for i in {1..100}; do
    (
      if "$BIN" pane send --scope session --text "ping $i" --yes >/dev/null 2>&1 && \
         "$BIN" pane list --session "$SESSION_NAME" --json >/dev/null 2>&1; then
        echo ok >>"$tmp"
      else
        echo fail >>"$tmp"
      fi
    ) &
  done
  wait
  local ok_count
  local fail_count
  ok_count=$(grep -c ok "$tmp" 2>/dev/null || true)
  fail_count=$(grep -c fail "$tmp" 2>/dev/null || true)
  rm -f "$tmp"
  echo "mixed-ok=$ok_count fail=$fail_count"
  if [[ $fail_count -ne 0 ]]; then
    fail_total=1
  fi
}

run_view_storm() {
  require_cmd jq
  echo "=== 6. View storm (${VIEW_STORM_PASSES}x${VIEW_STORM_PANES} panes, -P${VIEW_STORM_CONCURRENCY}) ==="
  local ids
  ids=$("$BIN" pane list --session "$SESSION_NAME" --json | jq -r '.data.panes[].id' | head -n "$VIEW_STORM_PANES")
  if [[ -z "$ids" ]]; then
    echo "No panes available for view storm" >&2
    fail_total=1
    return
  fi
  local fail=0
  for _ in $(seq 1 "$VIEW_STORM_PASSES"); do
    if ! printf "%s\n" "$ids" | xargs -n1 -P"$VIEW_STORM_CONCURRENCY" -I{} "$BIN" pane view --pane-id {} --rows 40 --cols 160 --mode ansi --yes >/dev/null 2>&1; then
      fail=$((fail + 1))
    fi
  done
  echo "view-fail=$fail"
  if [[ $fail -ne 0 ]]; then
    fail_total=1
  fi
}

run_output_flood() {
  echo "=== 7. Output flood (${OUTPUT_FLOOD_LINES} lines) ==="
  if ! "$BIN" pane run --pane-id "$PANE0_ID" --command "yes X | head -n ${OUTPUT_FLOOD_LINES}" --yes >/dev/null 2>&1; then
    echo "output flood failed" >&2
    fail_total=1
  fi
}

run_tool_loop() {
  if [[ "$RUN_TOOLS" != "1" ]]; then
    return
  fi
  echo "=== 8. Tool loop (Codex) ==="
  for i in {1..5}; do
    echo "--- Iteration $i ---"
    "$BIN" pane run --scope session --command "codex" --yes
    sleep 2
    "$BIN" pane run --scope session --command "send me some emoji flows" --yes --timeout 20s
    "$BIN" pane run --scope session --command "/exit" --yes
  done
}

run_scaleup() {
  echo "=== 9. Scale-up fan-out ==="
  for _ in {1..10}; do
    "$BIN" pane add --session "$SESSION_NAME" --yes
  done
  "$BIN" pane run --scope session --command "echo fanout-ok" --yes
}

scale_to_panes
run_snapshot_storm
run_parallel_snapshot
run_osc_flood
run_payload_send
run_mixed_fanout
run_view_storm
run_output_flood
run_view_storm
run_tool_loop
run_scaleup

if [[ $fail_total -ne 0 ]]; then
  echo "Stress test failures detected" >&2
  exit 1
fi

echo "All stress tests passed"
