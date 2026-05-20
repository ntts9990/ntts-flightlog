# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go CLI module for `ntts-flightlog`. The executable entrypoint lives in `cmd/flightlog/main.go`; command handling is under `internal/cli/`. Core packages are organized by responsibility: `internal/db/` for SQLite storage and migrations, `internal/metrics/` for report metrics, `internal/tui/` for Bubble Tea UI, `internal/worklog/` for worklog files, `internal/agent/` for agent detection, and `internal/migrate/` for legacy migration. End-to-end tests live in `e2e/`. Fixtures, golden files, and scenario inputs live in `testdata/`. Distribution assets are in `bin/`, `scripts/`, `.goreleaser.yml`, and `skill/ntts-flightlog/`.

## Build, Test, and Development Commands

- `go test ./...` runs all unit and integration tests.
- `go test ./... -race -count=1` matches the CI race-test path.
- `go build -o dist/flightlog ./cmd/flightlog` builds the local CLI.
- `CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=ci" -o dist/flightlog ./cmd/flightlog` matches CI cross-build flags.
- `go test ./internal/metrics/... -run TestGolden -count=1` checks golden report snapshots.
- `go test ./internal/db -run TestConcurrent -count=1 -v` runs the concurrent DB stress test.
- `go test ./internal/db -bench=BenchmarkColdOpen -benchtime=10x -run=^$` checks cold-open performance.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on edited `.go` files and keep imports normalized by `go test` or `goimports` if available. Package names are short lowercase nouns such as `cli`, `db`, and `metrics`. Test files use Go conventions: `*_test.go`, `TestName`, `BenchmarkName`, and table tests where cases clarify behavior. Keep CLI-facing text concise and preserve existing Korean output patterns where relevant.

## Testing Guidelines

Prefer focused package tests for behavior changes, then run `go test ./...` before handoff. Update `testdata/golden/` only when output changes are intentional. Add migration tests beside `internal/migrate/` and DB tests beside `internal/db/`; put full CLI workflows in `e2e/`.

## Commit & Pull Request Guidelines

Recent history uses imperative, phase-oriented subjects such as `Add Phase C: ...`. New commits must follow the repository Lore protocol: start with the intent, then include useful trailers such as `Constraint:`, `Rejected:`, `Confidence:`, `Scope-risk:`, `Directive:`, `Tested:`, and `Not-tested:`. Pull requests should describe user-visible behavior, list verification commands, link issues when available, and include screenshots or terminal output for TUI changes.

## Security & Configuration Tips

Do not commit generated runtime state from `.ntts-flightlog/`, `.omx/`, `.omc/`, `dist/`, or coverage files. Release automation uses GoReleaser and GitHub tokens; keep token use confined to CI secrets.
