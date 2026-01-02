# Performance Guide

Peaky Panes exposes a small set of performance knobs for dashboard previews. These affect **dashboard tiles and previews**, not the full terminal view (Ctrl+k).

## Summary

- **Presets**: control pane-view scheduling and throttling.
- **Render policy**: decide whether only visible panes update or all panes update.
- **Preview render mode**: choose cached vs direct vs off.

## Presets

Presets affect how often pane views are requested and how much work is done per refresh.

- **low**: battery saver, conservative scheduling.
- **medium**: balanced.
- **high**: smoother updates, higher CPU.
- **max**: no throttling. Highest CPU, default for new installs.
- **custom**: override specific fields under `dashboard.performance.pane_views`.

## Render policy

- **visible** (default): only panes currently visible on the dashboard update live.
- **all**: update every pane, even off-screen. This is heavy at scale.

## Preview render mode

- **cached**: uses ANSI cache + background refresh. Best overall cost/perf.
- **direct**: bypasses cache and renders synchronously. Smoothest, but CPU-heavy.
- **direct** (default): prefer this when you want maximum smoothness.
- **off**: disables live previews in the dashboard (reduces CPU). Ctrl+k still shows the live terminal.

## Recommended settings

- **Laptop / battery**:
  - preset: low
  - render_policy: visible
  - preview_render: cached

- **Balanced**:
  - preset: medium
  - render_policy: visible
  - preview_render: cached

- **High-end workstation**:
  - preset: max
  - render_policy: visible
  - preview_render: direct

## Example config

```yaml
dashboard:
  performance:
    preset: max            # low | medium | high | max | custom
    render_policy: visible # visible | all
    preview_render:
      mode: direct         # cached | direct | off
```

## Custom overrides

Use `preset: custom` to override specific values:

```yaml
dashboard:
  performance:
    preset: custom
    render_policy: visible
    preview_render:
      mode: direct
    pane_views:
      max_concurrency: 8
      max_inflight_batches: 4
      max_batch: 16
      min_interval_focused_ms: 1
      min_interval_selected_ms: 1
      min_interval_background_ms: 1
      timeout_focused_ms: 1000
      timeout_selected_ms: 800
      timeout_background_ms: 600
      pump_base_delay_ms: 1
      pump_max_delay_ms: 1
      force_after_ms: 1
      fallback_min_interval_ms: 1
```

## Notes

- `render_policy: all` and `preview_render: direct` are intentionally heavy.
- Dashboard previews are independent from the full terminal view.
- If you want perfectly smooth previews, prefer `preview_render: direct` and a higher preset.
