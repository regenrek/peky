## Strict implementation rules

- Build complete, production grade implementations that scale for >1000 users. No backwards compatibility/fallback concerns.
- Design for long term sustainability, maintainability, and reliability. no mvp/minimal implementation.
- Never create compatibility shims, no thin compat wrappers.
- Ship changes as the single canonical implementation in the primary codepath. Remove obsolete or legacy code in the same change.  
- Use direct, first class integrations. Replace any shims, glue code, thin wrappers, or transitional adapter layers with the real implementation at the correct layer.  
- Reduce tech debt as part of delivery by deleting dead paths, consolidating duplicated logic, and keeping one source of truth.  
- Keep a single source of truth for all business rules and policy. This includes validation, enums, flags, constants, and configuration.  
- Keep the UI as a pure view layer (thin clients) that renders backend models and options using API responses or shared types.
- Validate and sanitize all user-controlled inputs (paths, names) before OS/process calls.

# Workflow

1. Do not create git worktrees unless explicitly requested by the user. If so put them under peakypanes-worktress/<worktree-name>

# Testing

1. Maintain at least 60% code coverage for all Go modules to ensure reliability and early regression detection.
2. Cover critical native session manager behaviors with integration-style tests (real PTY or faithful harness).  
3. Test TUI state transitions and edge-case updates (resize, refresh failures, empty data).


# Coding Rules

Strict Go project rules (internal policy)

Versioning
- go.mod must set: go 1.25.x
- go.mod must set: toolchain go1.25.y (exact patch)
- Tools must be pinned (go.mod tool directive or equivalent); CI rejects unpinned tool updates

Repo layout (mandatory)
- cmd/<bin>/main.go only wiring + flags + start; no business logic
- internal/ contains all non-public code; pkg/ only if intended for external import
- testdata/ only for fixtures/golden files
- One module per repo by default; multi-module only with written rationale

Package rules
- Package names: single word, lower-case; no util/common/helpers packages
- No cyclic imports (CI check)
- internal/<domain>/... may not import internal/<other-domain>/... except through small interface packages (defined dependency directions)

File size (hard limits)
- Max 500 LOC per .go file (exclude blank lines + comments)
- Hard fail at 800 LOC
- Max 50 funcs/methods per file
- Max 15 exported identifiers per package (else split package or redesign API)

Function / complexity limits
- Max 60 LOC per function (exclude blank/comment)
- Max 6 parameters (use struct params beyond that)
- Cyclomatic complexity max 15 (CI via gocyclo or golangci-lint)
- No naked returns in funcs > 20 LOC

Error/logging rules
- No panic outside tests except truly unrecoverable init; must be justified in code comment
- All returned errors must include context; wrap with %w when propagating
- No log.Fatal/os.Exit outside cmd/
- Context required on any call that can block (I/O, RPC, DB); timeouts required at boundaries

Testing rules
- Every new package needs unit tests; every new exported symbol needs a test
- Unit tests: no network, no real DB, no wall clock sleeps
- Integration tests: build tag //go:build integration
- CI must run: go test ./... + go test -race ./...
- Package coverage gates:
  - domain packages >= 90%
  - others >= 80%
- Flaky test policy: any flake -> quarantine tag + fix before re-enable

Static checks (CI gates)
- gofmt (diff must be empty)
- go vet
- staticcheck
- govulncheck
- golangci-lint with configured set (incl gocyclo, errcheck, ineffassign, revive)

Enforcement
- Pre-commit hooks mirror CI
- CI is source of truth; PR cannot merge on red

Exceptions
- Allowed only with: short ADR + owner + expiry date
- CI allows exceptions only via explicit allowlist file reviewed by maintainers
