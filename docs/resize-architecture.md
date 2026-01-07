# Resize Architecture Map

## Flow (CLI → daemon → layout → PTY → TUI)

1) CLI/TUI issues layout ops (`pane.resize`, `pane.reset-sizes`, `pane.zoom`, `pane.split`, `pane.close`, `pane.swap`).
2) sessiond validates + routes ops to native manager.
3) native manager applies SSOT layout engine (`internal/layout`) and updates pane rects.
4) TUI rebuilds layout engines from session snapshots (`LayoutTree`) and renders the layout canvas.
5) Pane view requests (per visible pane) send cols/rows to sessiond; daemon calls `Window.Resize` before rendering.
6) PTY + VT resize happen inside `terminal.Window.Resize`, then frame render responds to the updated size.

## Mouse/Mode Inventory (UI vs Terminal focus)

UI mode (terminal focus off):
- Mouse routing: `updateDashboardMouse` → resize hover/drag → context menu → quick reply → scroll wheel → selection.
- Drag resize: left press on edge/corner starts drag, motion updates preview (mouse_throttle_ms), release commits, outside click cancels.
- Right-click (Button3): opens context menu on pane body.
- Cursor shapes: OSC 22 col/row/diag-resize on hover, pointer/text elsewhere.

Terminal mode (Ctrl+K):
- Mouse/keys pass through via `SendMouse` and key forwarding.
- Resize hit-zones stay UI-owned (mouse drag still works).
- Context menu disabled while terminal focus is active.

## Legacy/duplication audit

- SSOT layout engine is the only resize/retile path.
- Native pane dimensions always come from layout engine rects (`applyLayoutToPanes`).
- TUI layout canvas uses layout engine snapshots; no grid preview mode.
- No legacy retile code paths remain (retile helper removed; LayoutBaseSize is shared).
