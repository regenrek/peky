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
