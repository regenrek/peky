# Resize + Context Menu Release Notes

## Highlights
- UI-mode drag resize with snapping, hover/active guides, size labels, and cursor shapes.
- Keyboard resize mode (ctrl+shift+r by default; configurable) with nudges, edge cycling, snap toggle, reset sizes, zoom.
- Right-click context menu (Add Pane, Split Right/Down, Close, Zoom/Unzoom, Reset Sizes).
- CLI layout ops output can include `--json --after/--diff` layout trees.

## Known limitations
- WezTerm/iTerm2 parity not shipped yet (Ghostty-first). Capability matrix pending.

## Release checklist (resize)
- Run: `go test ./...`
- Run: `go test ./internal/layout -bench . -run ^$`
- Run (integration): `go test -tags=integration ./internal/sessiond`
- Verify UI mode in Ghostty with `right-click-action=context-menu` and `mouse-reporting=true`.
