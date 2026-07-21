<!-- file: .github/copilot-instructions.md -->
<!-- version: 1.1.0 -->
<!-- guid: a3f2e1d4-7b8c-4a9e-b5f6-2c3d4e5f6a7b -->
<!-- last-edited: 2026-07-21 -->

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
