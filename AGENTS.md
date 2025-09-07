# AGENTS

Guidelines for contributors and AI coding agents working in this repository.

## Goals

- Keep changes minimal, focused, and idiomatic to Go.
- Prefer root‑cause fixes over band‑aids; don’t refactor unrelated code.
- Maintain user‑facing behavior and CLI flags unless explicitly changing them.

## Project Snapshot

- Language: Go (modules enabled)
- Entry: `main.go`
- Core package: `runner/` (watcher, engine, flags, config, proxy)
- Docs: `README*.md`, `air_example.toml`, `docs/`
- Tooling: `Makefile`, `hack/check.sh`, `hooks/pre-commit`

## Quick Start

- Build: `make build`
- Install: `make install`
- CI setup + vendor: `make ci`
- Lint/format/check staged files: `make check`
- Run tests: `go test ./...`

Tip: run `make init` once to install `goimports`, set up `golangci-lint`, and enable the pre‑commit hook.

## Code Style (Go)

- Formatting: rely on `goimports` (invoked by `hack/check.sh`).
- Linting: fixes must satisfy `golangci-lint` (same config used in CI).
- Errors: wrap with context; return early; avoid panics in library code.
- Concurrency: use contexts where appropriate; avoid data races; prefer channels or mutexes over ad‑hoc globals.
- Logging: keep logs concise, actionable, and consistent with existing patterns.
- Public API/flags: changing behavior or flags requires README updates and tests.

## Tests

- Location: alongside code as `*_test.go` (see `runner/*_test.go`).
- Scope: unit tests near behavior changes; table‑driven where it fits.
- Run locally: `go test ./...`

## Common Change Points

- Config fields: edit `runner/config.go`, update parsing, defaults, and tests; reflect in `README.md` and `air_example.toml`.
- CLI flags: update `runner/flag.go`, sync help text, and README usage.
- Watcher behavior: `runner/watcher.go`; add tests for edge cases (create/delete/move events, include/exclude rules).
- Proxy/browser reload: `runner/proxy*.go`; keep docs aligned with README’s proxy section.

## Tooling and Commands

- `make init`: installs tooling and pre‑commit hook.
- `make check`: runs formatting + `golangci-lint` via `hack/check.sh`.
- `make build|install|ci`: standard workflows.
- Prefer `rg` for repository searches; keep file reads to small chunks.

## Documentation

- Update `README.md` for any user‑visible change (flags, config, examples).
- Keep examples in `air_example.toml` accurate when adding/removing fields.
- Include concise migration notes in PRs when behavior changes.

## PR & Commit Guidelines

- Commit messages: imperative, present tense, concise summary line.
- Include rationale and tradeoffs in the PR description.
- Link related issues; note breaking changes clearly.

## For AI Coding Agents (Codex CLI)

- Planning: for multi‑step tasks, maintain an explicit plan and mark progress.
- Edits: use a single focused patch per logical change; prefer `apply_patch`.
- Preambles: briefly state what you’re about to do before running commands.
- Searches: prefer `rg` and read files in ≤250‑line chunks.
- Validation: run `make check` and `go test ./...` to verify changes.
- Scope control: avoid touching unrelated files; don’t reformat the repo wholesale.
- No external network: assume offline; do not add dependencies lightly.

## Release Notes (maintainers)

- Follow the README “Release” section for tagging; CI performs builds.

---

Questions or uncertain scope? Open an issue or ask for clarification before implementing.
