# migrate-loop

Autonomous TDD migration harness driven by `claude -p`.

## What it does

Given a migration spec, `migrate-loop` runs a four-phase TDD loop:

1. **PLAN** — a planner agent writes a comprehensive *failing* test suite from the spec.
2. **CODE** — a coder agent iterates on the implementation, frozen out of test files, until everything is green or a budget is exhausted.
3. **COVER** — a coverage agent identifies and fills gaps in the migrated code.
4. **PR** — pushes the branch and opens a pull request.

If the harness gets stuck (tests look wrong, budget exhausted, or persistent stagnation), it writes `ESCALATION.md` to the worktree and exits 2 for a human to take over. Resume with `--resume`.

## Quickstart

```bash
go install github.com/jdfalk/migrate-loop/cmd/migrate-loop@latest
migrate-loop --spec mymigration.md --budget 50
```

## Documentation

- [USAGE.md](docs/USAGE.md) — flags, exit codes, spec format, escalation kinds.
- [design.md](docs/design.md) — full architecture, state machine, error handling, testing strategy.

## Status

Initial v1 — Go test runner only. Architecture admits other runners (`pytest`, `vitest`) via the `Runner` interface; impls are follow-up work.
