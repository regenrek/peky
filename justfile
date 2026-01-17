set shell := ["bash", "-cu"]

init:
  scripts/reinit.sh --all

build:
  @bash -cu 'set -euo pipefail; go install ./cmd/peky'

# Canonical local dev happy-path:
# - rebuild CLI/daemon (`go install`)
# - reset dev runtime dir under `/tmp/peky-dev-$uid`
# - restart daemon + open dashboard
devup: build
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    uid="${UID:-$(id -u)}"; devroot="/tmp/peky-dev-$uid"; \
    export PEKY_DATA_DIR="$devroot"; export PEKY_RUNTIME_DIR="$devroot"; export PEKY_CONFIG_DIR="$devroot"; \
    if [[ -e "$PEKY_DATA_DIR" ]]; then trash "$PEKY_DATA_DIR"; fi; \
    mkdir -p "$PEKY_DATA_DIR"; chmod 700 "$PEKY_DATA_DIR"; \
    : > "$PEKY_CONFIG_DIR/config.yml"; chmod 600 "$PEKY_CONFIG_DIR/config.yml"; \
    export PEKY_LOG_FILE="$PEKY_DATA_DIR/peky-dev.log"; \
    PEKY_FRESH_CONFIG=1 PEKY_LOG_LEVEL=debug PEKY_LOG_FORMAT=text PEKY_LOG_SINK=file \
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
