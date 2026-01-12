set shell := ["bash", "-cu"]

init:
  scripts/reinit.sh --all

build:
  @bash -cu 'set -euo pipefail; go install ./cmd/peky'

dev:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    mkdir -p "$devroot"; chmod 700 "$devroot"; \
    if [[ ! -f "$PEKY_CONFIG_DIR/config.yml" ]]; then : > "$PEKY_CONFIG_DIR/config.yml"; chmod 600 "$PEKY_CONFIG_DIR/config.yml"; fi; \
    export PEKY_LOG_FILE="$devroot/peky-dev.log"; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --keep-daemon; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    "$peky" start -y'

watch:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    mkdir -p "$devroot"; chmod 700 "$devroot"; \
    if [[ ! -f "$PEKY_CONFIG_DIR/config.yml" ]]; then : > "$PEKY_CONFIG_DIR/config.yml"; chmod 600 "$PEKY_CONFIG_DIR/config.yml"; fi; \
    export PEKY_LOG_FILE="$devroot/peky-dev.log"; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --keep-daemon; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    scripts/dev-watch -- --'

devfresh:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    mkdir -p "$devroot"; chmod 700 "$devroot"; \
    if [[ ! -f "$PEKY_CONFIG_DIR/config.yml" ]]; then : > "$PEKY_CONFIG_DIR/config.yml"; chmod 600 "$PEKY_CONFIG_DIR/config.yml"; fi; \
    export PEKY_LOG_FILE="$devroot/peky-dev.log"; \
    PEKY_FRESH_CONFIG=1 PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text \
    PEKY_LOG_SINK=file PEKY_LOG_FILE="$PEKY_LOG_FILE" \
    PEKY_LOG_ADD_SOURCE=1 PEKY_LOG_INCLUDE_PAYLOADS=1 \
    PEKY_PERF_TRACE_ALL=1 scripts/reinit.sh --all; \
    PEKY_FRESH_CONFIG=1 PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text \
    PEKY_LOG_SINK=file PEKY_LOG_FILE="$PEKY_LOG_FILE" \
    PEKY_LOG_ADD_SOURCE=1 PEKY_LOG_INCLUDE_PAYLOADS=1 \
    PEKY_PERF_TRACE_ALL=1 "$peky" start -y'

dev-fresh-all:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    if [[ -e "$PEKY_DATA_DIR" ]]; then trash "$PEKY_DATA_DIR"; fi; \
    mkdir -p "$PEKY_DATA_DIR"; chmod 700 "$PEKY_DATA_DIR"; \
    : > "$PEKY_CONFIG_DIR/config.yml"; \
    export PEKY_LOG_FILE="$PEKY_DATA_DIR/peky-dev.log"; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --all --no-daemon-restart; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    "$peky" daemon restart -y; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    "$peky" start -y'

dev-fresh-tracing: build
  @bash -cu 'set -euo pipefail; \
    export PEKY_TUI_TRACE_INPUT=1; \
    export PEKY_TUI_TRACE_INPUT_FILE="/private/tmp/tui-input-raw.log"; \
    export PEKY_TUI_TRACE_INPUT_REPAIRED=1; \
    export PEKY_TUI_TRACE_INPUT_REPAIRED_FILE="/private/tmp/tui-input-repaired.log"; \
    : > "$PEKY_TUI_TRACE_INPUT_FILE"; \
    : > "$PEKY_TUI_TRACE_INPUT_REPAIRED_FILE"; \
    gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    if [[ -e "$PEKY_DATA_DIR" ]]; then trash "$PEKY_DATA_DIR"; fi; \
    mkdir -p "$PEKY_DATA_DIR"; chmod 700 "$PEKY_DATA_DIR"; \
    : > "$PEKY_CONFIG_DIR/config.yml"; \
    export PEKY_LOG_FILE="$PEKY_DATA_DIR/peky-dev.log"; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --all --no-daemon-restart; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    "$peky" daemon restart -y; \
    PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
    PEKY_LOG_FILE="$PEKY_LOG_FILE" PEKY_LOG_ADD_SOURCE=1 \
    PEKY_LOG_INCLUDE_PAYLOADS=1 PEKY_PERF_TRACE_ALL=1 \
    "$peky" start -y'

dev-tmux-tracing: build
  @bash -cu 'set -euo pipefail; scripts/dev-tmux --tracing'

mark-scroll-start:
  @bash -cu 'set -euo pipefail; uid="${UID:-$(id -u)}"; log="/tmp/peky-dev-$uid/peky-dev.log"; \
    python3 -c '\''import datetime; print("MARK scroll_start ts="+datetime.datetime.now().astimezone().isoformat(timespec="seconds"))'\'' \
    | tee -a "$log" >/dev/null'

mark-scroll-stop:
  @bash -cu 'set -euo pipefail; uid="${UID:-$(id -u)}"; log="/tmp/peky-dev-$uid/peky-dev.log"; \
    python3 -c '\''import datetime; print("MARK scroll_stop ts="+datetime.datetime.now().astimezone().isoformat(timespec="seconds"))'\'' \
    | tee -a "$log" >/dev/null'

dev-fresh-daemon-restart:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    mkdir -p "$PEKY_DATA_DIR"; chmod 700 "$PEKY_DATA_DIR"; \
    if [[ ! -f "$PEKY_CONFIG_DIR/config.yml" ]]; then : > "$PEKY_CONFIG_DIR/config.yml"; fi; \
    export PEKY_LOG_FILE="$PEKY_DATA_DIR/peky-dev.log"; \
    "$peky" daemon restart -y'

prod:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    scripts/reinit.sh --all; \
    "$peky" start -y'
