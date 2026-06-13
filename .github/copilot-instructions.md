<!-- file: .github/copilot-instructions.md -->
<!-- version: 1.0.0 -->
<!-- guid: a3f2e1d4-7b8c-4a9e-b5f6-2c3d4e5f6a7b -->
<!-- last-edited: 2026-06-13 -->

# migrate-loop — Additional Context

Org-wide coding standards (file headers, language rules, commit format) are at
**https://github.com/falkcorp/.github** and apply automatically to this repo.

For full project context: **CLAUDE.md** at the repo root.

## Project overview

Autonomous TDD migration harness for multi-package refactors. Language: Go.

## Key directories

| Directory | Purpose |
|---|---|
| `cmd/migrate-loop/` | CLI entry point |
| `internal/agent/` | Agent orchestration |
| `internal/phases/` | Migration phase runners |
| `internal/runner/` | Test runner integration |
| `internal/spec/` | Migration spec parsing |
| `internal/state/` | State tracking across phases |
| `internal/worktree/` | Git worktree management |

## Build commands

```bash
make build        # Build binary → ./bin/migrate-loop
make test         # Run all tests with race detector
make test-live    # Run tests with live API tag
make lint         # go vet
make cover        # Test coverage report
make install      # Install to GOPATH/bin
```
