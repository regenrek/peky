# Investigation: Live Pane Preview Stalls

## Summary
Two contributing issues identified. First, snapshot preview caching originally refused dirty ANSI frames (fixed earlier). Second, UI event backpressure drops pane_updated events, so some panes never get a pane view refresh until a manual refresh (e.g., Ctrl+K). We are batching events and refreshing pane views in batches to prevent drops from starving updates.

## Symptoms
- Some visible dashboard panes show stale preview text while adjacent panes keep updating.
- Stale panes often have continuous output (long-running commands).
- Ctrl+K (terminal focus toggle) forces the previews to refresh.

## Investigation Log

### 2025-12-30 - Phase 1: Initial Assessment
**Hypothesis:** Preview updates are tied to window update events and might be throttled or skipped for certain panes.
**Findings:** Traced dashboard render path and preview sources. Dashboard uses PaneView when available and falls back to snapshot preview lines when PaneView is empty.
**Evidence:** internal/tui/views/render_dashboard_panes.go:345-352
**Conclusion:** Stale previews can originate from snapshot preview cache, not only PaneView.

### 2025-12-30 - Phase 2: Preview Cache and ANSI Cache Interaction
**Hypothesis:** Snapshot preview cache refuses to update when ANSI cache is dirty.
**Findings:** renderPreviewLines reads cached ANSI and returns lines even if ready is false, but snapshot logic gated updates on readiness. UpdateSeq can keep cacheDirty true during continuous output. refreshANSICache sets cacheDirty when UpdateSeq changes mid-render. rp-cli builder hung and was interrupted; continued manual trace.
**Evidence:** internal/native/pane_build.go:175-181; internal/native/session.go:297-335; internal/terminal/window_render.go:48-118; internal/terminal/window.go:450-458
**Conclusion:** For high-frequency panes, cacheDirty can stay true, so snapshot refused to commit preview lines, producing stale tiles.

### 2025-12-30 - Phase 3: Pane View Scheduling (Partial)
**Hypothesis:** Pane view requests are not scheduled for some visible panes.
**Findings:** Dashboard render always attempts PaneView first and otherwise uses snapshot preview. However, pane view updates depend on daemon events or full refresh.
**Evidence:** internal/tui/views/render_dashboard_panes.go:345-352; internal/tui/app/model_update.go:270-315
**Conclusion:** Pane view scheduling is present but can be starved by event drops.

### 2025-12-30 - Phase 4: Git History
**Hypothesis:** Recent performance changes may have introduced stricter gating.
**Findings:** Recent commits in this branch focus on pane view optimizations and authoritative render modes.
**Evidence:** git log -n 5 --oneline (3e71054, cb23a65)
**Conclusion:** Recent perf work intersects with preview behavior; no single commit isolated yet.

### 2025-12-30 - Phase 5: Snapshot Cache Fix
**Hypothesis:** Allowing dirty cached frames will keep previews updating under continuous output.
**Findings:** Snapshot now stores preview lines even when cache is dirty and tracks dirty state for follow-up refreshes. Integration test starts a busy PTY, waits for dirty cache, then asserts preview is populated.
**Evidence:** internal/native/session.go:295-335; internal/native/preview_cache.go:3-7; internal/native/snapshot_integration_test.go:1-62
**Conclusion:** Fix implemented; improves snapshot preview updates but does not solve all live pane stalls.

### 2025-12-30 - Phase 6: Event Backpressure
**Hypothesis:** UI event channel drops pane_updated events, so some panes never trigger pane view refresh.
**Findings:** UI logs show frequent "sessiond client: drop event type=pane_updated" while daemon shows no drops. This indicates client-side backpressure. Dropped events explain panes that stay stale until manual refresh.
**Evidence:** /tmp/peakypanes-ui.log (drop events), internal/sessiond/client.go:72-116
**Conclusion:** Event backpressure is a second root cause; pane view refresh must be batched/coalesced to avoid drop-induced stalls.

## Root Cause
Primary: UI event backpressure drops pane_updated events, so some panes never schedule pane view refreshes; these panes remain stale until a manual refresh. Secondary: snapshot preview cache previously refused dirty ANSI frames (now fixed).

## Recommendations
1. Batch/coalesce daemon events in the UI and refresh pane views in batches so no pane is starved by event drops.
2. Keep accepting dirty cached frames in snapshot previews (already implemented).
3. Add metrics/logging to flag sustained event drops and pane view starvation.

## Preventive Measures
- Regression tests for preview updates under constant output (PTY write loop + snapshot).
- Track event drop counts and surface warnings in debug mode.
