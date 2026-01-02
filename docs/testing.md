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
# Baseline run (10 panes, fresh config).
./scripts/perf-profiler \
  --layout testdata/performance-tests/peakypanes-perf10.yml \
  --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s

# Full trace + all panes live (tmux-like).
./scripts/perf-profiler \
  --layout testdata/performance-tests/peakypanes-perf10.yml \
  --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s \
  --trace-all --paneviews-all

# Live pprof server (dev-only; requires profiler build tag).
go build -tags profiler ./cmd/peakypanes
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

Perf-profiler outputs (per run under `.bench/profiler-12/<timestamp>/`):

- `report.txt`: run metadata + summaries
- `timings.tui_perf.txt`: event→req/resp and req→resp (UI pipeline)
- `timings.input_output.txt`: input→output (end-to-end)
- `timings.summary.txt`: staged startup timeline

When to use which mode:

- **Baseline** (default): visible panes only; measure realistic dashboard behavior.
- **`--paneviews-all`**: sets dashboard.performance.render_policy=all to render all panes live for profiling.
- **`--trace-all`**: logs every pane view timing to compute accurate averages.

GitHub Actions runs gofmt checks, go vet, go test with coverage, and race on Linux.
