<!-- file: CLAUDE.md -->
<!-- version: 1.0.0 -->
<!-- guid: c9e7b3a2-4d5f-4e6a-8b1c-3f2e1d0a9b8c -->
<!-- last-edited: 2026-06-13 -->

# migrate-loop

Autonomous TDD migration harness for multi-package refactors.

## Coding Standards

Org-wide coding standards are in the `.standards/` git submodule (cloned from `https://github.com/falkcorp/.github`).
Always clone with `git clone --recurse-submodules` so these are available.

Key files:
- **File headers (MANDATORY):** `.standards/instructions/file-headers.md`
- **Go rules:** `.standards/instructions/go.md`
- **Commit format:** `.standards/instructions/commit-messages.md`

## Build & Test Commands

```bash
make build        # Build binary → ./bin/migrate-loop
make test         # Run all tests with race detector
make test-live    # Run tests with live API tag
make lint         # go vet
make cover        # Test coverage report
make install      # Install to GOPATH/bin
```

## Architecture

`migrate-loop` orchestrates multi-package Go refactors via a phase-based pipeline:

- `cmd/migrate-loop/` — CLI entry point
- `internal/agent/` — Agent orchestration
- `internal/phases/` — Migration phase runners (plan → apply → verify)
- `internal/runner/` — Test runner integration (`go test -race ./...`)
- `internal/spec/` — Migration spec parsing (YAML)
- `internal/state/` — State tracking across phases
- `internal/worktree/` — Git worktree management per migration task
