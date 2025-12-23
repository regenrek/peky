# Changelog

All notable changes to this project will be documented in this file.
This format is based on Keep a Changelog.

## Unreleased

## 0.0.3 - 2025-12-23

### Added
- Sidebar icon system with size/ASCII fallbacks (`PEAKYPANES_ICON_SET`, `PEAKYPANES_ICON_SIZE`).
- Command palette action for creating windows.
- Dashboard keymap overrides via `dashboard.keymap` in the global config.

### Changed
- Sidebar hierarchy styling (single caret, per-session spacing, no "Windows"/"Sessions" labels).
- Pane preview tiles use collapsed borders with consistent shared edges and highlight colors.
- Tab/shift+tab now cycles panes across windows; footer help reflects new navigation.
- Navigation uses ctrl+W/A/S/D for project/session selection to keep quick reply always active.
- Window list toggle key moved to ctrl+u to avoid conflicts.
- Preview header line removed and global header spacing added for cleaner layout.
- Theme uses design tokens for consistent TUI colors.
- Dashboard attach behavior is configurable (current terminal, new terminal, or detached).
- Default attach behavior now opens sessions in a new terminal to avoid switching other tmux clients.

### Fixed
- Active/target tile borders now draw fully even when sharing edges.
- Pane selection no longer jumps back during rapid tab cycling.
- Session creation no longer switches other tmux clients when launched outside tmux.

## 0.0.2 - 2025-12-23

### Added
- Agent state detection for Codex CLI and Claude Code TUI (optional hook scripts + `dashboard.agent_detection` toggles).
- CI workflow (gofmt check, go vet, go test + coverage, race, tmux integration tests on Linux).
- Opt-in tmux integration test (`PEAKYPANES_INTEGRATION=1`) for session lifecycle coverage.
- CLI/dashboard argument parsing tests plus small-terminal render coverage.

### Changed
- npm Windows packages are temporarily disabled due to npm spam-detection blocks.
- Quick reply bar is always visible and the target pane is highlighted in the preview.
- Quick reply input is always active; `enter` sends (or attaches if empty), `esc` clears.
- Navigation and hotkeys now use `ctrl+` bindings to avoid input collisions (`ctrl+h/j/k/l`, `ctrl+g`, etc.).
- Pane preview tiles now truncate long lines and trim trailing blanks for consistent sizing.
- Pane preview footer no longer shows the window bar or path/layout/status lines.
- Quick reply input width now clamps to the available bar width.

### Fixed
- Quick reply sends now submit in Codex/Claude TUIs by sending literal text plus Enter.

## 0.0.1 - 2025-12-22

### Added
- npm distribution with per-platform optional dependency packages and a tiny launcher.
- `peakypanes popup` and `peakypanes dashboard --popup` for tmux popup dashboards (with window fallback).
- `peakypanes dashboard --tmux-session` to host the dashboard in a dedicated tmux session.
- `peakypanes setup` to check tmux availability and print install tips.
- Dashboard command palette (Ctrl+P) for quick actions.
- Rename session/window flows in the dashboard.

### Changed
- `peakypanes` now opens the dashboard directly (use `peakypanes dashboard --tmux-session` for the hosted session flow).
- Default layout bind key `prefix+g` now opens the popup dashboard.
- Module path standardized to `github.com/regenrek/peakypanes`.

### Fixed
- Tmux bind-key actions now preserve quoted arguments when applying layout key bindings.
