## Strict implementation rules

1. Build complete, production grade implementations that scale for 1000 plus users.  
2. Design for long term sustainability, maintainability, and reliability.  
3. Ship changes as the single canonical implementation in the primary codepath. Remove obsolete or legacy code in the same change.  
4. Preserve every existing feature and every UI option. Keep the UX surface intact. When wiring is incomplete, keep the UI visible and back it with clearly labeled stubs until the backend support is finished.  
5. Use direct, first class integrations. Replace any shims, glue code, thin wrappers, or transitional adapter layers with the real implementation at the correct layer.  
6. Reduce tech debt as part of delivery by deleting dead paths, consolidating duplicated logic, and keeping one source of truth.  
7. Keep a single source of truth for all business rules and policy. This includes validation, enums, flags, constants, and configuration.  
8. Keep the UI as a pure view layer that renders backend models and options using API responses or shared types.
9. Use `context` timeouts for external I/O and process execution to prevent hangs.  
10. Wrap errors with actionable context and preserve causes (`%w`).  
11. Validate and sanitize all user-controlled inputs (paths, names) before OS/process calls.

# Workflow

1. Do not create git worktrees unless explicitly requested by the user.

# Testing

1. Maintain at least 60% code coverage for all Go modules to ensure reliability and early regression detection.
2. Cover critical native session manager behaviors with integration-style tests (real PTY or faithful harness).  
3. Test TUI state transitions and edge-case updates (resize, refresh failures, empty data).
