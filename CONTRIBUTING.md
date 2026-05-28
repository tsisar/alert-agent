# Contributing to Alert Agent

Thanks for your interest in contributing. This document covers what you need
to know to get set up, work on a change, and submit it.

## Project status

Alert Agent is in **beta**. The HTTP API, CLI flags, and DB schema may still
change. Pin to a tagged release for production use.

## Getting started

### Prerequisites

- **Go 1.25+** (`go.mod` is the source of truth)
- **`templ`** for HTML template generation — installed automatically by
  `make install-templ`
- Optional: **Docker** for image builds, **Redis** for cross-restart dedup
  and queue testing

### One-time setup

```bash
git clone https://github.com/tsisar/alert-agent.git
cd alert-agent

cp .env.example .env
# fill in LLM_API_KEY at minimum; see configuration.md
$EDITOR .env
```

Configure MCP servers in `.mcp.json` (defaults to `mcp-grafana.example.com` —
point it at your own MCP server).

### Build, run, test

```bash
make generate    # regenerate templ code (.templ → *_templ.go)
make run         # run locally (depends on generate)
make test        # unit tests with coverage
make lint        # golangci-lint
make build       # production binary in bin/
```

`make generate` is implicit in `run`, `dev`, and `build` targets — you only
need it manually after editing `.templ` files in `internal/web/templates/`.

## Branch and commit conventions

### Branches

Branch off `main` (`release` for hotfixes only). Use kebab-case after the
prefix.

| Prefix      | Branched from | Purpose                                 |
|-------------|---------------|-----------------------------------------|
| `feature/`  | `main`      | New functionality                       |
| `fix/`      | `main`      | Bug fix targeting the next release      |
| `chore/`    | `main`      | Tooling, deps, refactors, no behavior   |
| `ci/`       | `main`      | CI / build pipeline changes             |
| `docs/`     | `main`      | Documentation only                      |
| `refactor/` | `main`      | Internal restructuring without behavior |
| `hotfix/`   | `release`     | Urgent production fix                   |

Examples: `feature/redis-streams-dlq`, `fix/ci-remove-trivy`,
`chore/bump-go-1.25`.

One logical scope per branch / PR. Don't bundle unrelated changes — if a
follow-up belongs to the same scope as an open PR, push to that branch rather
than opening a new one.

### Commit messages — Conventional Commits

We follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

Format: `<type>[optional scope]: <description>`, with an optional body and
footer separated by blank lines.

```
feat: add hat wobble

Optional body explaining motivation, separated by a blank line.

Refs: #42
```

Types:

| Type       | When to use                                       |
|------------|---------------------------------------------------|
| `feat`     | New feature (MINOR in SemVer)                     |
| `fix`      | Bug fix (PATCH in SemVer)                         |
| `docs`     | Documentation only                                |
| `style`    | Formatting, missing semicolons — no logic change  |
| `refactor` | Neither a fix nor a feature                       |
| `perf`     | Performance improvement                           |
| `test`     | Adding or updating tests                          |
| `build`    | Build system or external dependency changes       |
| `ci`       | CI/CD configuration changes                       |
| `chore`    | Routine tasks, maintenance                        |
| `revert`   | Reverting a previous commit                       |

Rules:

- Description follows `type:` directly — no capital letter, no trailing period.
- Imperative mood: "add feature", not "added feature".
- Keep the first line under 72 characters.
- Blank line between title and body.
- **No `Co-Authored-By` lines.**

Breaking changes are marked with `!` after the type, or with a
`BREAKING CHANGE:` footer:

```
feat!: drop support for Go 1.20

feat(api)!: rename Users endpoint to Accounts
```

### English only

All code comments, godoc, `.md` files, commit messages, and identifier names
are in English. PR descriptions and inline review comments too. Chat with
maintainers can be in any language, but anything that ends up in the repo
must be in English.

## Pull requests

- Open against `main` (`release` for hotfixes).
- Reference the issue you are addressing in the PR description if there is one.
- CI must be green before review.
- Update `CHANGELOG.md` under `## [Unreleased]` if the change is user-visible.
  Entries are written for the user — purely internal refactors do not belong
  there.
- For UI changes, attach a screenshot or a short clip.

## Testing

- Add unit tests for new logic in the package you change. Test files sit
  alongside production code as `*_test.go`.
- Run `make test` locally before pushing. CI runs the same target.
- For changes touching the webhook, MCP client, or notifiers, run `make
  test-webhook` against a local instance and verify the end-to-end path
  manually.

## Code style

We follow [Google's Go Style Guide](https://google.github.io/styleguide/go/)
and the [Standard Go Project Layout](https://github.com/golang-standards/project-layout).
Mechanical formatting is enforced by tooling:

- `gofmt -s` / `goimports` clean. `make format` does this for you.
- `golangci-lint` clean. `make lint` runs it.

Project-specific conventions:

- Default to **no comments** unless the *why* is non-obvious. Identifier
  names should do the work; godoc on exported symbols is the exception.
- Use `fmt.Errorf("…: %w", err)` for wrapping — `%w` at the end. Don't log
  and return the same error; do one or the other.
- Define interfaces in the package that **consumes** them, not in the
  package that implements them. Keep interfaces small (1–3 methods).
- Avoid package-level mutable state in library code. Pass dependencies via
  constructors.
- No backwards-compatibility shims or dead code. If something is unused,
  delete it.

See [`CLAUDE.md`](CLAUDE.md) for the complete agent-facing rule set
(includes the same conventions plus a few extra notes for AI assistants).

## Releasing

Release tagging is documented in [`docs/release-process.md`](docs/release-process.md).
Only maintainers cut releases — contributors don't need to worry about it.

## Reporting security issues

**Don't** open public issues for security vulnerabilities. See
[`SECURITY.md`](SECURITY.md) for the disclosure process.

## Code of conduct

Be kind. Assume good intent. Disagree about ideas, never about people.
