# Release Process

This document is the single source of truth for releasing Peaky Panes.

## Preconditions

- You are on `main` with a clean working tree.
- You have push access to the GitHub repo.
- Releases, Homebrew tap updates, and npm publishing are done by GitHub Actions; no local npm login required.

## Required Tests (must pass)

Run these before any release (matches CI):

```bash
unformatted=$(git ls-files -z '*.go' | xargs -0 gofmt -l)
if [ -n "$unformatted" ]; then
  echo "gofmt needed on:"
  echo "$unformatted"
  exit 1
fi

go vet ./...
go test ./... -coverprofile=cover.out
go tool cover -func=cover.out | tail -n 1
go test ./... -race
```

## Release Steps

1) Pick the next version

2) Update docs:
- Move `CHANGELOG.md` “Unreleased” → the new version with today’s date.
- Ensure `README.md` matches current behavior.
- Update any versioned examples (e.g. `scripts/fresh-run X.Y.Z`).

3) Commit and tag:

```bash
git add -A
git commit -m "release: vX.Y.Z"
git tag vX.Y.Z
git push origin main --tags
```

4) GitHub Actions publishes everything:

- Tag push triggers the `release` workflow, which runs GoReleaser to create the GitHub release + upload assets, then updates the Homebrew tap formula.
- Publishing the GitHub Release triggers the `npm Release` workflow, which builds npm packages from the GitHub release assets and publishes them using OIDC.

5) Verify installs (recommended):

```bash
brew install --build-from-source regenrek/tap/peakypanes
brew test regenrek/tap/peakypanes
```

6) Publish npm packages (GitHub Actions, Trusted Publishing):

- Creating/publishing the GitHub Release triggers the `npm Release` workflow, which builds the npm packages from the GitHub release assets and publishes all 5 packages using OIDC.
- Monitor the run under GitHub Actions → `npm Release`.

## Release Helper (recommended)

You can use the scripted helper to run the local parts (tests, tag, push) and trigger the GitHub Actions release pipeline:

```bash
scripts/release X.Y.Z
```

Dry run (no tag/push/release/publish, tests still run):

```bash
scripts/release X.Y.Z --dry-run
```

The helper requires a clean working tree and push access to the repo.

## Post-Release Verification

```bash
npm view peakypanes
npx -y peakypanes
```

## Notes

- The `release` workflow builds binaries into `dist/` and creates the GitHub release.
- The `release` workflow updates the Homebrew tap formula (`regenrek/homebrew-tap`) via `scripts/update-homebrew-tap`.
- `npm run build:npm-packages` copies those binaries into `packages/` and writes the npm metadata (used by the `npm Release` workflow).
- The meta package (`packages/peakypanes`) must be published last so it can resolve optional dependencies.
