# ADR-0001: Go Toolchain Version Exception

**Date:** 2025-12-28  
**Status:** Accepted  
**Owner:** kregenrek  
**Expiry:** 2026-03-31

## Context

Project policy requires `go 1.25.x` and `toolchain go1.25.y` in `go.mod`.
The repo currently pins `go 1.24.2` and `toolchain go1.24.11`.
We have not yet validated Go 1.25 across all local dev and CI environments for this repo.

## Decision

Keep the module and toolchain pinned to Go 1.24.x temporarily.
This exception is time-boxed and will be removed once Go 1.25 is validated in CI and local tooling.

## Consequences

- Builds continue using Go 1.24.x until the upgrade is verified.
- The ADR must be removed after the upgrade to Go 1.25.x is completed.
