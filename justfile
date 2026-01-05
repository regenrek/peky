set shell := ["bash", "-cu"]

init:
  scripts/reinit.sh --all

dev:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE=/tmp/peakypanes-dev.log PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --all; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE=/tmp/peakypanes-dev.log PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    "$peak" start -y'

watch:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE=/tmp/peakypanes-dev.log PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/reinit.sh --all; \
    PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text PEAKYPANES_LOG_SINK=file \
    PEAKYPANES_LOG_FILE=/tmp/peakypanes-dev.log PEAKYPANES_LOG_ADD_SOURCE=1 \
    PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 PEAKYPANES_PERF_TRACE_ALL=1 \
    scripts/dev-watch -- --'

devfresh:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    PEAKYPANES_FRESH_CONFIG=1 PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text \
    PEAKYPANES_LOG_SINK=file PEAKYPANES_LOG_FILE=/tmp/peakypanes-dev.log \
    PEAKYPANES_LOG_ADD_SOURCE=1 PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 \
    PEAKYPANES_PERF_TRACE_ALL=1 scripts/reinit.sh --all; \
    PEAKYPANES_FRESH_CONFIG=1 PEAKYPANES_LOG_LEVEL=debug PEAKYPANES_LOG_FORMAT=text \
    PEAKYPANES_LOG_SINK=file PEAKYPANES_LOG_FILE=/tmp/peakypanes-dev.log \
    PEAKYPANES_LOG_ADD_SOURCE=1 PEAKYPANES_LOG_INCLUDE_PAYLOADS=1 \
    PEAKYPANES_PERF_TRACE_ALL=1 "$peak" start -y'

dev-fresh-all:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; peky="$gobin/peky"; \
    export PEAKYPANES_DATA_DIR=/tmp/peakypanes-dev; \
    export PEAKYPANES_RUNTIME_DIR=/tmp/peakypanes-dev; \
    export PEAKYPANES_CONFIG_DIR=/tmp/peakypanes-dev; \
    rm -rf "$PEAKYPANES_DATA_DIR"; \
    mkdir -p "$PEAKYPANES_DATA_DIR"; \
    chmod 700 "$PEAKYPANES_DATA_DIR"; \
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
    export PEAKYPANES_DATA_DIR=/tmp/peakypanes-dev; \
    export PEAKYPANES_RUNTIME_DIR=/tmp/peakypanes-dev; \
    export PEAKYPANES_CONFIG_DIR=/tmp/peakypanes-dev; \
    if [[ ! -f "$PEAKYPANES_CONFIG_DIR/config.yml" ]]; then : > "$PEAKYPANES_CONFIG_DIR/config.yml"; fi; \
    export PEAKYPANES_LOG_FILE="$PEAKYPANES_DATA_DIR/peakypanes-dev.log"; \
    "$peky" daemon restart -y'

prod:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    scripts/reinit.sh --all; \
    "$peak" start -y'
