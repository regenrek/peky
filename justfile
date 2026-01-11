set shell := ["bash", "-cu"]

init:
  scripts/reinit.sh --all

dev:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peakypanes-dev-$uid"; \
    export PEAKYPANES_DATA_DIR="$devroot"; export PEAKYPANES_RUNTIME_DIR="$devroot"; export PEAKYPANES_CONFIG_DIR="$devroot"; \
    mkdir -p "$devroot"; chmod 700 "$devroot"; \
    if [[ ! -f "$PEAKYPANES_CONFIG_DIR/config.yml" ]]; then : > "$PEAKYPANES_CONFIG_DIR/config.yml"; chmod 600 "$PEAKYPANES_CONFIG_DIR/config.yml"; fi; \
    export PEAKYPANES_LOG_FILE="$devroot/peakypanes-dev.log"; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --keep-daemon; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    "$peak" start -y'

watch:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peakypanes-dev-$uid"; \
    export PEAKYPANES_DATA_DIR="$devroot"; export PEAKYPANES_RUNTIME_DIR="$devroot"; export PEAKYPANES_CONFIG_DIR="$devroot"; \
    mkdir -p "$devroot"; chmod 700 "$devroot"; \
    if [[ ! -f "$PEAKYPANES_CONFIG_DIR/config.yml" ]]; then : > "$PEAKYPANES_CONFIG_DIR/config.yml"; chmod 600 "$PEAKYPANES_CONFIG_DIR/config.yml"; fi; \
    export PEAKYPANES_LOG_FILE="$devroot/peakypanes-dev.log"; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --keep-daemon; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/dev-watch -- --'

devfresh:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peakypanes-dev-$uid"; \
    export PEAKYPANES_DATA_DIR="$devroot"; export PEAKYPANES_RUNTIME_DIR="$devroot"; export PEAKYPANES_CONFIG_DIR="$devroot"; \
    mkdir -p "$devroot"; chmod 700 "$devroot"; \
    if [[ ! -f "$PEAKYPANES_CONFIG_DIR/config.yml" ]]; then : > "$PEAKYPANES_CONFIG_DIR/config.yml"; chmod 600 "$PEAKYPANES_CONFIG_DIR/config.yml"; fi; \
    export PEAKYPANES_LOG_FILE="$devroot/peakypanes-dev.log"; \
    PEAKYPANES_FRESH_CONFIG=1 PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text \
    PEAKYPANES_LOG_SINK=file PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" \
    PEAKYPANES_LOG_ADD_SOURCE=1 PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 \
    PEAKYPANES_PERF_TRACE_ALL=1 scripts/reinit.sh --all; \
    PEAKYPANES_FRESH_CONFIG=1 PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text \
    PEAKYPANES_LOG_SINK=file PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" \
    PEAKYPANES_LOG_ADD_SOURCE=1 PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 \
    PEAKYPANES_PERF_TRACE_ALL=1 "$peak" start -y'

dev-fresh-all:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peakypanes-dev-$uid"; \
    export PEAKYPANES_DATA_DIR="$devroot"; export PEAKYPANES_RUNTIME_DIR="$devroot"; export PEAKYPANES_CONFIG_DIR="$devroot"; \
    if [[ -e "$PEAKYPANES_DATA_DIR" ]]; then trash "$PEAKYPANES_DATA_DIR"; fi; \
    mkdir -p "$PEAKYPANES_DATA_DIR"; chmod 700 "$PEAKYPANES_DATA_DIR"; \
    : > "$PEAKYPANES_CONFIG_DIR/config.yml"; \
    export PEAKYPANES_LOG_FILE="$PEAKYPANES_DATA_DIR/peakypanes-dev.log"; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --all --no-daemon-restart; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    "$peky" daemon start -y; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    "$peak" start -y'

dev-fresh-tracing: build
  @bash -cu 'set -euo pipefail; \
    export PEAKYPANES_TUI_TRACE_INPUT=1; \
    export PEAKYPANES_TUI_TRACE_INPUT_FILE="/private/tmp/tui-input-raw.log"; \
    export PEAKYPANES_TUI_TRACE_INPUT_REPAIRED=1; \
    export PEAKYPANES_TUI_TRACE_INPUT_REPAIRED_FILE="/private/tmp/tui-input-repaired.log"; \
    : > "$PEAKYPANES_TUI_TRACE_INPUT_FILE"; \
    : > "$PEAKYPANES_TUI_TRACE_INPUT_REPAIRED_FILE"; \
    gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peakypanes-dev-$uid"; \
    export PEAKYPANES_DATA_DIR="$devroot"; export PEAKYPANES_RUNTIME_DIR="$devroot"; export PEAKYPANES_CONFIG_DIR="$devroot"; \
    if [[ -e "$PEAKYPANES_DATA_DIR" ]]; then trash "$PEAKYPANES_DATA_DIR"; fi; \
    mkdir -p "$PEAKYPANES_DATA_DIR"; chmod 700 "$PEAKYPANES_DATA_DIR"; \
    : > "$PEAKYPANES_CONFIG_DIR/config.yml"; \
    export PEAKYPANES_LOG_FILE="$PEAKYPANES_DATA_DIR/peakypanes-dev.log"; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --all --no-daemon-restart; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    "$peky" daemon start -y; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE="$PEAKYPANES_LOG_FILE" PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    "$peak" start -y'

dev-fresh-daemon-restart:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peakypanes-dev-$uid"; \
    export PEAKYPANES_DATA_DIR="$devroot"; export PEAKYPANES_RUNTIME_DIR="$devroot"; export PEAKYPANES_CONFIG_DIR="$devroot"; \
    mkdir -p "$PEAKYPANES_DATA_DIR"; chmod 700 "$PEAKYPANES_DATA_DIR"; \
    if [[ ! -f "$PEAKYPANES_CONFIG_DIR/config.yml" ]]; then : > "$PEAKYPANES_CONFIG_DIR/config.yml"; fi; \
    export PEAKYPANES_LOG_FILE="$PEAKYPANES_DATA_DIR/peakypanes-dev.log"; \
    "$peky" daemon restart -y'

prod:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    scripts/reinit.sh --all; \
    "$peak" start -y'
