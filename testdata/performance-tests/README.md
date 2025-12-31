# Performance Layouts

This folder contains the canonical layouts used for baseline performance runs.

## Layouts

- `peakypanes-perf10-control.yml`
  - 10 panes, shell-only baseline (no Claude).
  - Use this to measure pure pane startup + UI refresh.
- `peakypanes-perf10.yml`
  - 10 panes, runs `claude` in each pane and sends a single prompt.
  - Use this to measure Claude startup and steady-state updates.

## Recommended baseline runs

```bash
./scripts/perf-profiler --layout testdata/performance-tests/peakypanes-perf10-control.yml --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s
./scripts/perf-profiler --layout testdata/performance-tests/peakypanes-perf10.yml --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s
```

Each run writes outputs under `.bench/profiler-12/<timestamp>/` (already gitignored).
Key artifacts:
- `report.txt` (run metadata + timing summary)
- `timings.tsv` (per-pane timings)
- `timings.summary.txt` (min/max/avg in ms)
- `cpu.pprof`, `heap.pprof`, `fgprof.pprof`, `trace.out`

## Notes

- Keep `send_delay_ms` at 10s in the Claude layout for stable baselines.
- Use the control layout to isolate pane startup vs external process startup.
- If a run shows `sessions=0 panes=0` in `daemon.log`, the session failed to start (baseline invalid).
