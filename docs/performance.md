# Performance

This repo includes a set of reproducible performance/stress tools for peky.

## Goals (multipane-first)

We optimize for extreme fan-out workloads (hundreds of panes) where the daemon must remain responsive:

- Bounded memory growth per pane (no unbounded scrollback growth; no runaway caches).
- Low allocation rate on hot paths (PTY output ingest, view rendering, snapshotting).
- Predictable latency under load (view storms, focus churn, broadcast sends).
- Graceful degradation under abuse (bounded concurrency, backpressure, clear errors).

## Local benchmarks (micro)

Run the Go microbenchmarks and save results to `.bench/`:

```bash
scripts/perf-bench
```

Current benchmark areas:
- `internal/terminal`: frame cache and snapshot rendering path.
- `internal/sessiond`: pane view response construction.
- `internal/vt`: scrollback push/reflow behaviors.

## Profile a real session (macro)

Run a fresh session with the profiler build tag enabled and capture CPU/heap profiles:

```bash
scripts/perf-profiler
```

This captures a run directory under `.bench/` with:
- daemon logs
- pprof profiles (cpu/heap; optional block/mutex/trace/fgprof)
- a summary report

Useful flags:
- `--layout <file>`: choose a workload layout
- `--secs <n>`: CPU profile seconds
- `--trace <n>` / `--fgprof <n>`: capture deeper traces sequentially
- `--paneviews-all`: render all panes (worst-case dashboard)

## Generate perf layouts

Create reproducible perf layouts and run them via `peky start --path`:

```bash
scripts/perf-12pane
scripts/perf-40pane
```

Tune:
- `GRID_ROWS`, `GRID_COLS`: pane count
- `RATE_MS`: output cadence per pane
- `ANSI_HEAVY=1`: use heavier ANSI output

## Compare against tmux

Use a similar workload under tmux for input->output timing checks:

```bash
scripts/perf-tmux --panes 40 --mode ping
```

## Minimum acceptance (release)

For releases, we expect:
- `scripts/cli-stress.sh` passes locally on a typical dev machine.
- No sustained unbounded RSS growth when running `scripts/perf-40pane` for several minutes.
- `scripts/perf-bench` does not regress materially vs the previous release tag (compare `.bench/bench-*.txt`).
