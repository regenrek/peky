---
name: run-dev
description: Launch the PeakyPanes dev workflow by running scripts/dev-run in the current repo. Use when the user says "run dev" or asks to open a fresh Ghostty dev session that go installs, restarts the daemon, and launches peakypanes.
---

# Run Dev

## Overview
Run the repo's `scripts/dev-run` to open a fresh Ghostty window that installs the CLI, restarts the daemon, and launches PeakyPanes.

## Workflow
1) Ensure the current working directory is the PeakyPanes repo root (look for `scripts/dev-run`).
2) If not in the repo, ask for the correct path or `cd` into it.
3) Run `scripts/dev-run` with any user-provided args (e.g. `scripts/dev-run start --layout dev-3`).
4) If Ghostty is missing or the OS is not macOS, report that `scripts/dev-run` will fail and ask how to proceed.

## Notes
- Default to no args if the user just says "run dev".
- If the user wants a specific layout or command, append those args to `scripts/dev-run`.
