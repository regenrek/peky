# Testing

Run unit tests with coverage:

```bash
go test ./... -coverprofile /tmp/peakypanes.cover
go tool cover -func /tmp/peakypanes.cover | tail -n 1
```

Race tests:

```bash
go test ./... -race
```

Manual npm smoke run (fresh HOME/XDG config):

```bash
scripts/fresh-run
scripts/fresh-run X.Y.Z --with-project
```

Profiling (dev-only):

```bash
# Build with profiler tag to enable pprof HTTP server + fgprof.
go build -tags profiler ./cmd/peakypanes

# Baseline run (10 panes).
./scripts/perf-profiler --layout testdata/performance-tests/peakypanes-perf10.yml --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s

# Render all panes (dev-only; requires perf debug).
PEAKYPANES_PERF_PANEVIEWS_ALL=1 PEAKYPANES_PERF_DEBUG=1 ./scripts/perf-profiler --layout testdata/performance-tests/peakypanes-perf10.yml --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s

# Live pprof server (dev-only).
./peakypanes daemon --pprof
```

For a live TUI viewer, use `pproftui` against the daemon:

```bash
pproftui -refresh 2s http://127.0.0.1:6060/debug/pprof/profile
```

tmux comparison (run from Ghostty for a 10-pane baseline):

```bash
./scripts/perf-tmux --panes 10
```

GitHub Actions runs gofmt checks, go vet, go test with coverage, and race on Linux.
