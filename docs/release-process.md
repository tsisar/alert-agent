# Release Process

This document describes how new versions of `alert-agent` are released. Draft — open for feedback.

## 1. Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html):

```
MAJOR.MINOR.PATCH
```

- **MAJOR** — incompatible API/configuration changes (breaking changes)
- **MINOR** — backwards-compatible new features
- **PATCH** — backwards-compatible bug fixes

Tag prefix: `v` (e.g. `v1.2.0`).

Pre-releases use a suffix: `v1.2.0-rc.1`, `v1.2.0-beta.2`.

## 2. Branching strategy

We use a release branch flow.

### Permanent branches

| Branch    | Purpose                                                           |
|-----------|-------------------------------------------------------------------|
| `main`    | Active development. All feature branches merge here.              |
| `release` | Stable code, ready to ship. Tags are created only on this branch. |

### Temporary branches

| Prefix      | Purpose                            | Branched from | Merged into        |
|-------------|------------------------------------|---------------|--------------------|
| `feature/*` | New functionality                  | `main`        | `main`             |
| `fix/*`     | Bug fix targeting the next release | `main`        | `main`             |
| `hotfix/*`  | Urgent production fix              | `release`     | `release` + `main` |

### Flow diagram

```
feature/* ──► main ──(stabilization)──► release ──(tag vX.Y.Z)──► production
                                              ▲
hotfix/*  ────────────────────────────────────┘
              │
              └──► main (back-merge)
```

## 3. Preparing a release

### 3.1. Get `main` ready

1. All planned MRs are merged into `main`.
2. CI on `main` is green.
3. The `## [Unreleased]` section in `CHANGELOG.md` reflects everything in this release.

### 3.2. Stabilization period

Open a release MR `main` → `release`:

- title: `release: vX.Y.Z`
- no new features during the stabilization window — only bug fixes
- bug fixes land on `main` first and are cherry-picked into the release MR (or made directly off the release
  branch when needed)

### 3.3. Version bump

In a dedicated commit on the release MR:

- update `CHANGELOG.md`: rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`
- add the corresponding compare link at the bottom of the file
- (no source code edit needed — the version is injected at build time via `-ldflags` from the git tag)

Commit message format: `chore(release): vX.Y.Z`

### 3.4. Merge and tag

1. Merge the release MR into `release` (no squash — preserve history).
2. Tag the head of `release`:

   ```bash
   make release VERSION=vX.Y.Z
   ```

   The target switches to `release`, pulls, validates (branch, working tree, tag format/conflicts, sync with origin),
   creates an annotated tag, and pushes it.

3. The tag pipeline publishes the Docker image and creates a GitLab Release automatically.

### 3.5. Back-merge into `main`

After tagging, merge `release` back into `main` so the CHANGELOG bump lands in active development:

```bash
git checkout main
git merge --no-ff release
git push
```

## 4. Hotfix process

When a critical bug is found in production:

1. Create `hotfix/<short-desc>` off `release` (not off `main`!).
2. Apply the fix and update `CHANGELOG.md`.
3. Bump the PATCH version: `vX.Y.Z` → `vX.Y.(Z+1)`.
4. Open an MR into `release`.
5. After merge, tag the new patch release.
6. **Always** back-merge `release` → `main`, otherwise the fix will be lost in the next release.

## 5. CHANGELOG

Format: [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/).

`CHANGELOG.md` lives at the repo root. Every MR that changes user-visible behavior adds an entry under
`## [Unreleased]`.

Fixed categories:

- **Added** — new functionality
- **Changed** — change in existing behavior
- **Deprecated** — marked for removal
- **Removed** — removed functionality
- **Fixed** — bug fixes
- **Security** — security fixes

Entries are written for the user, not the developer. Internal refactors with no behavioral impact do not belong
in the CHANGELOG.

## 6. Release notes

The tag pipeline extracts the matching `## [X.Y.Z]` heading section from
`CHANGELOG.md` and publishes it as the release notes via the platform's
Releases API (a `curl` call against the GitLab Releases API in the previous
GitLab CI; under GitHub this will be the GitHub Releases API):

- name: `vX.Y.Z`
- description: the matching `## [X.Y.Z]` section is extracted from
  `CHANGELOG.md` at pipeline time. The extractor is heading-based — the
  section must start with exactly `## [X.Y.Z]` for it to be picked up.
- assets: link to the Docker image (extend with binaries / helm chart as
  needed).

## 7. Artifacts

The CI pipeline produces different image tags depending on the trigger:

| Trigger           | Image tag(s)          | Purpose                                     |
|-------------------|-----------------------|---------------------------------------------|
| Push to `main`    | `:<short-sha>`        | Development build (pushed; staging/testing) |
| Push to `release` | `:rc-<short-sha>`     | Release candidate (pushed)                  |
| Tag `vX.Y.Z`      | `:vX.Y.Z` + `:latest` | Stable production release                   |
| Tag `vX.Y.Z-rc*`  | `:vX.Y.Z-rc.N`        | Pre-release (does **not** touch `:latest`)  |

Rules:

- The `vX.Y.Z` tag is immutable — it always points to the same commit and the same binary.
- `:latest` is updated **only** by stable tags. It is not touched by pushes to `main` or `release`.
- The version is baked into the binary at build time via `-ldflags` (`internal/version` package). At runtime
  it is exposed by `alert-agent version`.

## 8. Rollback

- Artifacts are immutable — rollback is a redeploy of an earlier image from the registry.
- Database migrations follow the expand/contract pattern: ship a release that understands both old and new
  schemas first, then ship a release that cleans up the old shape.

## 9. Release checklist

- [ ] CI on `main` is green
- [ ] `CHANGELOG.md` `[Unreleased]` is populated and meaningful
- [ ] Release MR `main` → `release` is open
- [ ] `[Unreleased]` renamed to `[X.Y.Z] - YYYY-MM-DD`, compare link added
- [ ] Release MR CI is green
- [ ] Merged into `release`
- [ ] Tag `vX.Y.Z` created and pushed
- [ ] Release notes auto-created from CHANGELOG by the tag pipeline
- [ ] Docker image `:vX.Y.Z` (and `:latest` for stable tags) is published
- [ ] Deployed to staging
- [ ] Deployed to production
- [ ] `release` back-merged into `main`

## 10. Anti-patterns (don't do this)

- ❌ Tag directly off `main`
- ❌ Overwrite an existing tag (`git tag -f`)
- ❌ Force-push to `release`
- ❌ Mix new features and fixes in a hotfix release
- ❌ Release without updating the CHANGELOG
- ❌ Forget to back-merge `release` → `main` after a hotfix

---

> **TODO** (review during editing):
> - decide on the support policy for previous MAJOR versions (LTS or not)
> - decide who is allowed to merge into `release` (CODEOWNERS / approval rules)
> - extend the GitLab Release `assets` block once we publish binaries / helm charts