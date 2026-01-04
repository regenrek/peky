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

prod:
  @bash -cu 'set -euo pipefail; gobin="${GOBIN:-$(go env GOPATH)/bin}"; peak="$gobin/peakypanes"; \
    scripts/reinit.sh --all; \
    "$peak" start -y'
