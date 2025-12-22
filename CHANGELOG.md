# Changelog

All notable changes to this project will be documented in this file.
This format is based on Keep a Changelog.

## Unreleased

### Changed
- npm Windows packages are temporarily disabled due to npm spam-detection blocks.

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
