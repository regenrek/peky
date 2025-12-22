# Release Process

This document is the single source of truth for releasing Peaky Panes.

## Preconditions

- You are on `main` with a clean working tree.
- You have push access to the GitHub repo.
- You are logged into npm (`npm whoami`).

## Required Tests (must pass)

Run these before any release:

```bash
go test ./...
```

## Release Steps

1) Pick the version (e.g. `0.1.0`).

2) Update docs:
- Move `CHANGELOG.md` “Unreleased” → the new version with today’s date.
- Ensure `README.md` matches current behavior.

3) Commit and tag:

```bash
git add -A
git commit -m "release: vX.Y.Z"
git tag vX.Y.Z
git push origin main --tags
```

4) Build and publish the GitHub release (GoReleaser):

```bash
goreleaser release --clean
```

5) Generate npm packages from `dist/`:

```bash
npm run build:npm-packages
```

6) Publish npm packages (platform packages first, then the meta package):

Note: Windows npm packages are currently disabled due to npm spam-detection blocks.

```bash
cd packages/peakypanes-darwin-arm64 && npm publish --access public
cd ../peakypanes-darwin-x64 && npm publish --access public
cd ../peakypanes-linux-arm64 && npm publish --access public
cd ../peakypanes-linux-x64 && npm publish --access public
cd ../peakypanes && npm publish --access public
```

## Release Helper (recommended)

You can use the scripted helper to run the full flow (tests, tag, release, npm publish):

```bash
scripts/release X.Y.Z
```

Dry run (no tag/push/release/publish, tests still run):

```bash
scripts/release X.Y.Z --dry-run
```

The helper requires:
- clean working tree
- `CHANGELOG.md` contains the target version
- npm auth (`npm whoami` must succeed)

## Post-Release Verification

```bash
npm view peakypanes
npx -y peakypanes setup
```

## Notes

- `goreleaser release --clean` builds binaries into `dist/` and creates the GitHub release.
- `npm run build:npm-packages` copies those binaries into `packages/` and writes the npm metadata.
- The meta package (`packages/peakypanes`) must be published last so it can resolve optional dependencies.
