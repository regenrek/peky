# Perf Summary (2025-12-31)

## Scope
Baseline vs latest perf-all (render all panes) after TUI scheduling fixes and perf-trace cleanup.

## Key runs
- Baseline: `.bench/profiler-12/20260101-070726`
- Latest (perf-all): `.bench/profiler-12/20260101-131328`

## Results (high level)
- **Event → request latency** improved from ~266 ms p50 (baseline) to **<200 ms** (no slow logs in latest).
- **Event → response tail** (30–60s in baseline) was an artifact of off-screen churn and stale perf tracing; now **no slow event→resp logs** with perf-all + tracer fix.
- **Render latency (request → response)** remains fast (~3–5 ms) and was never the bottleneck.

## Conclusions
- The performance issue was **TUI scheduling/visibility**, not daemon rendering.
- With all panes rendered (`PEAKYPANEVIEWS_ALL=1`), the live update path is stable and fast.
- The long-tail metrics are now accurate after clearing stale pending events in the perf tracer.

## Notes
- Dashboard normally renders only visible panes; off-screen panes won’t update live unless `PEAKYPANEVIEWS_ALL=1` is enabled for dev/profiling.

## Peakypanes vs tmux (2026-01-01)
Latest Peakypanes run (perf-profiler, trace-all) shows much lower UI update latency than Ghostty/tmux:

- **Peakypanes event → response**: avg **5.548 ms** (min 1.102, max 121.853)  
- **Peakypanes event → request**: avg **2.504 ms** (min 0.463, max 56.425)  
- **Peakypanes request → response**: avg **3.044 ms** (min 0.486, max 93.569)
- **Peakypanes input → output** (`since_input`): avg **17.315 ms** (min 9.804, max 27.874)

Ghostty/tmux baseline (output-change mode):

- **Input → output-change**: avg **135.8 ms** (min 92, max 294)

This indicates Peakypanes’ UI update path is faster than tmux under the same 10‑pane workload and scripted input.

## How to reproduce (Peakypanes vs tmux)

### Peakypanes (all panes live, full trace)

Run:

```bash
./scripts/perf-profiler \
  --layout testdata/performance-tests/peakypanes-perf10.yml \
  --secs 30 --fgprof 30 --trace 10 --gops --start-timeout 20s \
  --trace-all --paneviews-all
```

Results:

```bash
run=$(ls -td .bench/profiler-*/*/ | head -n1)
cat "$run/timings.tui_perf.txt"       # event→req/resp + req→resp (UI pipeline)
cat "$run/timings.input_output.txt"   # input→output (end-to-end)
```

Notes:
- `--trace-all` logs every pane view request/response timing (no sampling).
- `--paneviews-all` forces all panes live (tmux‑like behavior).
- The run uses `--fresh-config` with isolated runtime/config dirs.

### tmux/Ghostty (baseline)

Run:

```bash
./scripts/perf-tmux --layout testdata/performance-tests/peakypanes-perf10.yml --panes 10
```

This reports `input → output-change` latency for each pane under Ghostty + tmux.

## What these numbers mean

- **event → request / response** (Peakypanes only): internal TUI pipeline latency after output exists.
- **input → output** (Peakypanes `since_input`): end‑to‑end latency from sending input to first output (includes the external process).
- **input → output-change** (tmux): end‑to‑end latency in Ghostty + tmux.

## Interpretation

- Peakypanes’ **UI pipeline is significantly faster** than tmux in this workload.
- **External process startup (claude / shell)** dominates overall time to first output (~4–6s); Peakypanes cannot reduce that without changing spawn behavior.
