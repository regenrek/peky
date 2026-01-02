# Changelog

All notable changes to this project will be documented in this file.
This format is based on Keep a Changelog.

## Unreleased

### Added
- Performance profiler harness (`scripts/perf-profiler`) with trace/fgprof/gops capture and timing summaries.
- Performance layout fixtures for 10-pane baselines plus wait-for-output variant.
- Performance test README in `testdata/performance-tests/`.
- Performance tuning guide (`docs/performance.md`).
- `max` performance preset plus configurable dashboard preview render mode (`dashboard.performance.preview_render.mode`).

### Changed
- Layout send actions default to waiting for first output when `send_delay_ms` is omitted.
- Pane dimension limits now clamp across sessiond, terminal, and VT layers to prevent oversize allocations.
- VT parser data buffer reduced from 4MB to 64KB (configurable via env override).
- Pane view scheduling now prioritizes visible panes with per-pane in-flight tracking to reduce refresh overhead.
- Dashboard performance menu now surfaces render-policy + preview-render tradeoffs with preset hints.

### Fixed
- Pane view cache now enforces TTL + max-entry eviction to prevent unbounded growth.
- Dashboard pane view refresh no longer churns on off-screen panes, reducing update storms under load.

## 0.0.8 - 2025-12-31

### Fixed
- Claude quick reply submission now sends text and submit separately with a short delay so Enter triggers send instead of inserting a newline.

## 0.0.7 - 2025-12-30

### Added
- Configurable quit behavior (`dashboard.quit_behavior`) with prompt/keep/stop options for handling running sessions on exit.
- Quit confirmation dialog (shown only when panes are running) with an option to stop the daemon and kill all panes.

### Fixed
- Quick reply now detects Codex panes by title when command metadata is missing, so sends use bracketed paste + submit.
- Pane titles ignore glyph-only window titles and fall back to pane index with short IDs for disambiguation.

## 0.0.6 - 2025-12-30

### Added
- Pane view update sequencing with NotModified responses to skip unchanged renders.
- Pane view priority scheduling to keep focused panes responsive under load.
- VT damage tracking primitives for future incremental rendering work.
- Performance tooling: `scripts/perf-bench`, `scripts/perf-12pane`, and `scripts/perf-40pane`.
- Snapshot integration coverage for dirty ANSI cache previews.
- Pane view scheduler tests for starvation and timestamp preservation.
- Daemon profiling hooks for CPU/heap captures via `PEAKYPANES_CPU_PROFILE` and `PEAKYPANES_MEM_PROFILE` (build tag `profiler`).
- Quick reply slash commands that mirror command palette actions.
- Quick reply broadcast support with `/all` and optional scope targets.
- Quick reply history navigation with up/down cycling.
- Command registry to keep palette and slash actions in sync.
- Dashboard `pane_navigation_mode` config to choose spatial or memory selection behavior.
- Command palette entry for the `/all` broadcast quick reply.

### Changed
- Pane view rendering now respects client deadlines and can fall back to cached views under pressure.
- Lipgloss rendering snapshots VT cells outside the terminal lock to reduce input stalls.
- ANSI view rendering is cached-first with background refresh to avoid sync render spikes.
- TUI pane view cache stores update sequences to avoid re-requesting unchanged panes.
- Preview cache now keys on pane update sequence for deterministic refresh behavior.
- Snapshot previews now accept dirty ANSI frames and track dirty state for follow-up refreshes.
- Pane view NotModified gating uses ANSI cache sequence to avoid stalling live previews.
- TUI batches daemon events to refresh pane views without starving updates under load.
- Perf load script now defaults to 12 panes and lives at `scripts/perf-12pane`.
- Command palette actions now render from the shared registry.
- Quick reply help/hints now surface slash commands and history.
- Quick reply slash commands now show a padded dropdown overlay above the input with arrow-key selection and tab completion.
- Dashboard keymap defaults now use `ctrl+q/ctrl+e` for project nav, `ctrl+a/ctrl+d` for pane nav, and `ctrl+,` for edit config.
- Project and dashboard left/right navigation now defaults to spatial row selection (set `pane_navigation_mode: memory` to restore per-project memory).

### Removed
- Retired perf and investigation docs (`docs/perf.md`, `docs/investigation-live-preview.md`).

### Fixed
- Pane swaps no longer notify while holding the manager lock, preventing deadlocks.
- Pane view cache state is cleared on daemon restart to avoid stale seq comparisons.
- Live pane previews remain current even when ANSI cache lags or event throughput spikes.

## 0.0.5 - 2025-12-29

### Added
- Native multiplexer manager with PTY/VT panes and full-screen TUI support.
- Live pane rendering in the dashboard and project views for native sessions.
- Terminal focus toggle for native panes (default `ctrl+k`, configurable).
- Native mouse support in the dashboard: single-click selects panes, double-click toggles terminal focus, and motion forwarding is throttled to avoid CPU spikes.
- Mouse wheel scrollback for native panes with shift/ctrl modifiers and drag-to-select copy mode in terminal focus.
- Mouse drag selection now auto-copies to the clipboard and shows a success toast.
- Project config change detection to refresh selection without reopening projects.
- Pane management actions: add pane (split), move pane to new window, swap panes, and close pane with a running-process confirmation.
- Session-only jump keys (`alt+w` / `alt+s`) alongside the flat session/window navigation.
- Native pane scrollback and copy mode (selection + yank) with configurable key bindings.
- Scrollback reflow on resize for native panes.
- Smoke/integration coverage for native scrollback/copy, VT alt-screen + reflow, and key TUI state transitions.
- Daemon runtime state persistence (`state.json`) with automatic restore on startup.
- CLI and TUI daemon restart commands with confirmation.
- Per-pane restore failure tracking surfaced in snapshots.
- Command palette action to close all projects with a bulk close/kill confirmation.
- Dashboard empty-state splash with centered logo and quick open/help hint.
- ANSI Regular logo wordmark for the splash screen with block centering.

### Changed
- Native-only sessions; tmux UI integration and commands are removed.
- Layouts now use native split/grid definitions only (no tmux layout options or bind keys).
- Removed the `peakypanes pipe` streaming helper and tmux streaming layer.
- Project view navigation: `ctrl+w` / `ctrl+s` now cycles sessions and windows in a single vertical list.
- Add pane now auto-places into the grid without a direction prompt.
- Refactored command palette items for clarity.
- Window rendering now supports scrollback viewports and copy-mode highlights for native panes.
- Alternate screen panes no longer record scrollback history.
- Normal-screen mouse wheel always scrolls host scrollback (app mouse reporting is ignored outside alt-screen).
- Mouse motion forwarding now enables during drag selection even when the app isn't requesting motion events.
- Default pane titles now compress path-like window names to readable repo-relative labels and de-duplicate duplicates.
- Session env overrides are persisted and reapplied on restore; splits inherit session env.
- State persistence is debounced and flushed on shutdown to reduce write amplification.
- Terminal-focus tiles use a distinct border accent while focused.
- Header now uses a right-aligned logo and removes the left brand label.

### Removed
- Dashboard thumbnails row and related config options.

### Fixed
- Space key now passes correctly in native terminal focus.
- Command palette and project picker filters now reset on open/selection.
- Scrollback view stays anchored when new output arrives while scrolled up.
- Native manager no longer panics on shutdown when pane updates arrive after close.
- Input to closed PTYs now yields a friendly "pane closed" toast, marks panes dead, and prevents repeated send errors.
- Project tabs and pane lists no longer reorder during rapid navigation thanks to deterministic ordering.
- Stale refresh results are ignored so fast navigation cannot apply out-of-order snapshots.
- Terminal focus no longer exits on Escape, so ESC passes through to in-pane TUIs.
- Terminal focus pane rendering now preserves ANSI colors by honoring the client color profile.
- Pane view refreshes are serialized and queued to avoid daemon socket write timeouts under rapid updates.
- Refresh snapshots no longer time out behind pane view rendering thanks to a dedicated pane view connection.
- Daemon transport now shuts down dead client links and client writes honor request deadlines to prevent refresh timeouts.
- Client calls now guard against closed connections during daemon restarts.
- Terminal focus footer hint now highlights only when active.
- Mouse wheel scrolling no longer injects raw SGR mouse escape sequences into shells.

## 0.0.4 - 2025-12-25

### Added
- Dashboard tab now renders per-project columns with pane blocks and multi-line previews.
- Dashboard pane blocks show bordered tiles with per-pane status and metadata.
- Peek selected pane in a new terminal (default `ctrl+y`, configurable via `dashboard.keymap.peek_pane`).
- Persist hidden projects in `dashboard.hidden_projects` so closed projects stay out of tabs.
- Command palette entries to reopen hidden projects (and auto-unhide when opening from picker).

### Changed
- Dashboard navigation uses up/down to move panes within a project column and tab/shift+tab to switch columns; help/footer text updated.
- Dashboard refresh now captures pane previews for all running sessions (minimum 10 lines per pane).
- Close project now hides it from tabs instead of killing sessions (with optional kill in the dialog).

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
