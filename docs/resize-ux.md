# TUI Resize + Context Menu UX Spec

Scope
- Resize handles always active in the project layout canvas, regardless of terminal focus.
- Terminal focus (Ctrl+K) forwards keys to the pane; context menu remains UI-only.
- Quick reply passthrough: when the input is empty, terminal-like navigation keys are forwarded to the selected pane for interactive prompts.
- Ghostty-first; use standard mouse/cursor protocols (OSC 22 cursor shapes).

Definitions
- Layout space: normalized coordinate space (0..LayoutBaseSize) used by layout engine.
- Preview space: TUI preview rect in character cells.
- Edge: split boundary between two adjacent panes.
- Corner: intersection of vertical + horizontal split edges.

Resize Handles
- Active in project preview (always-on layout canvas).
- Edge hit zone: 1 cell thick line; hover target expands to 2 cells for safety.
- Corner hit zone: 2x2 cells around edge intersections; takes priority over edge hits.
- Cursor shapes:
  - vertical edge: col-resize (ew-resize).
  - horizontal edge: row-resize (ns-resize).
  - corner: nwse-resize or nesw-resize depending on geometry.

Drag Semantics
- Press left mouse on edge/corner to begin resize.
- Target edge locked for duration of drag.
- Drag preview updates at <=60Hz, last-move-wins.
- mouse_apply: live sends resize ops during drag; commit sends one op on release.
- Live mode throttles remote updates via mouse_throttle_ms.
- Escape cancels preview, restores prior layout.
- Release commits resize op to SSOT engine.
- Click outside while dragging cancels (treated as Esc).

Keyboard Resize Mode
- Toggle resize mode: `ctrl+r` (global in dashboard), exits with `esc`.
- When active: overlay cheat sheet + active edge highlight.
- Keys:
  - Arrow keys: nudge active edge by 10 units (grid step).
  - Shift+Arrow: coarse nudge (25 units).
  - Ctrl+Arrow: max nudge (50 units).
  - Tab: cycle edges in session.
  - Esc: exit resize mode.
  - `s`: toggle snapping on/off for current drag session.
  - `0`: reset sizes in session.

Snapping
- Snap targets (ordered):
  1) Equalize siblings.
  2) Common ratios: 1/2, 1/3, 2/3, 1/4, 3/4.
  3) Grid step alignment (layout units).
  4) Neighbor edge alignment (exact boundary).
- Threshold: 12 layout units (scaled by preview size).
- Hysteresis: 6 layout units.
- Modifier to disable snap: hold Alt while dragging.

Min Pane Size Policy
- Hard min size in layout units:
  - min width: 40
  - min height: 20
- Enforced by validator/normalizer in layout engine.

Visuals
- Hover edge highlight: thin line in Accent.
- Active edge highlight: solid line in AccentAlt.
- Guide lines during drag for affected edge(s).
- Size label near cursor: shows resulting % split and approx cols/rows.
- Optional freeze_content_during_drag reduces live preview churn.

Context Menu (Right-Click on pane body)
- Location: at cursor; clamp to viewport.
- Items:
  - Add Pane → Split Right
  - Add Pane → Split Down
  - Close Pane
  - Zoom / Unzoom
  - Reset Sizes (session)
- Keyboard navigation: Up/Down, Enter to select, Esc to dismiss.
- No focus change until action commits.

Defaults
- Snap enabled by default.
- Grid step: 10 layout units.
- Resize mode off by default.

Ghostty Notes
- Right-click action should be set to context menu in Ghostty config.
- Mouse reporting enabled (default). UI mode uses OSC 22 cursor shapes.

Ghostty reference (src/config/Config.zig)
- `right-click-action` default: `context-menu` (other values: paste/copy/copy-or-paste/ignore).
- `mouse-reporting` default: true (apps can request mouse reporting).
