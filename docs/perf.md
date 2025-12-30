# Performance measurement

This repo now ships two ways to measure performance changes:

1) **Microbenchmarks** (repeatable, fast)
2) **40‑pane load run** (end‑to‑end behavior)

## 1) Microbenchmarks

Run the built-in benchmark suite:

```bash
scripts/perf-bench
```

Output goes to `.bench/bench-<timestamp>.txt`.

What the benchmarks cover:

- `internal/terminal`:
  - ANSI render cost (`refreshANSICache`, i.e. `term.Render()`)
  - Lipgloss render cost (cursor + non-cursor)
- `internal/sessiond`:
  - Pane view response path (`NotModified`, `ANSI`, `Lipgloss`)

### Comparing runs

If you want to compare two result files, use `benchstat` if you have it installed:

```bash
benchstat .bench/bench-old.txt .bench/bench-new.txt
```

## 2) 40‑pane load run (tmux‑style)

Generate a temporary 40‑pane layout that continuously prints output:

```bash
scripts/perf-40pane
```

For ANSI-heavy output (more realistic render stress):

```bash
ANSI_HEAVY=1 scripts/perf-40pane
```

Then run:

```bash
peakypanes start --path .bench/perf-40
```

The layout defaults to a 5x8 grid with ~50ms output cadence per pane. You can override the grid and cadence via env vars:

```bash
GRID_ROWS=4 GRID_COLS=10 RATE_MS=25 ANSI_HEAVY=1 scripts/perf-40pane
```

### Measuring CPU/RAM

In a separate terminal, sample CPU + RAM usage (macOS example):

```bash
ps -o pid,%cpu,%mem,rss,command -p "$(pgrep -x peakypanes | head -n1)"
```

Repeat this a few times to estimate average usage under load.

### Comparing to tmux

To compare with tmux, create a similar 40‑pane workload in tmux and sample CPU/RAM the same way. The key is to keep:

- same pane count
- same output cadence
- same terminal size

This gives you a like‑for‑like comparison of steady‑state CPU cost.
