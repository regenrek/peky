set shell := ["bash", "-cu"]

init:
  scripts/reinit.sh --all

build:
  @bash -cu 'set -euo pipefail; go install ./cmd/peky'

# Uninstall the dev version of peky (removes the Go-installed binary)
# Also stops the daemon and removes all peky data for a fresh start
# This allows the npm global version to take precedence
uninstall-dev:
  @echo "Uninstalling dev peky from Go bin..."
  @gobin="${GOBIN:-$(go env GOPATH)/bin}"; peky="$gobin/peky"; \
    if [[ -f "$peky" ]]; then \
      echo "Trashing: $peky"; \
      trash "$peky"; \
      echo "✓ Dev peky uninstalled"; \
    else \
      echo "No dev peky found at $peky"; \
    fi; \
    echo ""; \
    echo "Stopping any running peky daemons..."; \
    if command -v peky >/dev/null 2>&1; then \
      peky daemon stop --yes >/dev/null 2>&1 || true; \
    fi; \
    os="$$(uname -s)"; \
    runtime_dir="$HOME/.config/peky"; \
    if [[ "$$os" == "Darwin" ]]; then \
      runtime_dir="$HOME/Library/Application Support/peky"; \
    fi; \
    pidpath="$$runtime_dir/daemon.pid"; \
    if [[ -f "$$pidpath" ]]; then \
      pid="$$(cat "$$pidpath" 2>/dev/null || true)"; \
      if [[ -n "$$pid" ]] && kill -0 "$$pid" 2>/dev/null; then \
        kill "$$pid" 2>/dev/null || true; \
        sleep 1; \
        kill -9 "$$pid" 2>/dev/null || true; \
        echo "✓ Daemon stopped (PID: $$pid)"; \
      fi; \
    fi; \
    pkill -9 "peky daemon" 2>/dev/null || true; \
    echo "✓ All peky daemon processes killed"; \
    echo ""; \
    echo "Removing peky data/config directories for fresh start..."; \
    os="$$(uname -s)"; \
    targets=(); \
    if [[ "$$os" == "Darwin" ]]; then \
      targets+=("$HOME/Library/Application Support/peky" "$HOME/.config/peky"); \
    else \
      targets+=("$HOME/.local/share/peky" "$HOME/.config/peky"); \
    fi; \
    removed_any="false"; \
    for target in "$${targets[@]}"; do \
      if [[ -e "$$target" ]]; then \
        trash "$$target"; \
        echo "✓ Trashed: $$target"; \
        removed_any="true"; \
      fi; \
    done; \
    if [[ "$$removed_any" != "true" ]]; then \
      echo "No data directories found"; \
    fi; \
    echo ""; \
    echo "The npm global version is now at:"; \
    echo "  /opt/homebrew/bin/peky"; \
    echo ""; \
    echo "To use it immediately, run:"; \
    echo "  export PATH=\"/opt/homebrew/bin:\$PATH\""

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
