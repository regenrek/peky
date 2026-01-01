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
# Baseline run (10 panes).
./scripts/perf-profiler --layout testdata/performance-tests/peakypanes-perf10.yml --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s

# Render all panes (dev-only; requires perf debug).
PEAKYPANES_PERF_PANEVIEWS_ALL=1 PEAKYPANES_PERF_DEBUG=1 ./scripts/perf-profiler --layout testdata/performance-tests/peakypanes-perf10.yml --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s
```

GitHub Actions runs gofmt checks, go vet, go test with coverage, and race on Linux.
