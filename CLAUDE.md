<!-- file: CLAUDE.md -->
<!-- version: 1.1.0 -->
<!-- guid: c9e7b3a2-4d5f-4e6a-8b1c-3f2e1d0a9b8c -->
<!-- last-edited: 2026-07-21 -->

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


## 📝 Changelog & TODO — Use the Fragment System (MANDATORY)

**Do not hand-edit `CHANGELOG.md`, and do not add new tasks straight into the
`TODO.md` inbox.** Both files are assembled from per-change fragments so that
parallel PRs never collide on them.

- **`CHANGELOG.md` is assembled, not hand-edited.** Add a fragment under
  `changelog.d/` (run `scriv create`, or write the Markdown file by hand). The
  fragments are folded into `CHANGELOG.md` at release time by `scriv`, and a CI
  check (`changelog-check.yml`) requires one on each PR. See `changelog.d/README.md`.
- **New `TODO.md` tasks are added via fragments.** Drop a Markdown fragment in
  `todo.d/` (see `todo.d/README.md`) instead of editing the `## 📥 Inbox`
  section. `scripts/assemble_todo.py` folds fragments in daily. This is
  **add-only**: checking a task off or removing it is a normal direct edit of
  `TODO.md`.
