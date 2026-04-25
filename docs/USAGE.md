# Usage

## Quickstart

```bash
migrate-loop --spec mymigration.md --budget 50
```

Drives an autonomous TDD migration:

1. **PLAN** — planner agent writes a comprehensive failing test suite from the spec.
2. **CODE** — coder agent iterates until tests pass, blocked from editing test files.
3. **COVER** — coverage agent identifies and fills gaps.
4. **PR** — pushes the branch and (if `gh` is installed) opens a PR.

If the harness gets stuck, it writes `ESCALATION.md` and exits 2.

## Spec format

YAML frontmatter + free-form markdown body. Required keys:

```markdown
---
title: my migration
slug: my-migration
target_packages:
  - internal/foo
  - internal/bar
test_runner: "go test -race -json ./..."
prior_examples:                 # optional — paths or PR refs
  - docs/specs/prior.md
success_criteria:               # optional — for human reviewers
  - all tests in target_packages pass
---

# Free-form prose

[narrative description, edge cases, gotchas]
```

See [`design.md`](design.md) §"Spec format" for the full schema.

## Resume after escalation

If the harness escalates (`ESCALATION.md` is written, exit 2), fix the issue
and resume from where it left off:

```bash
# Edit FROZEN_TESTS.md, ESCALATION.md, or the tests as needed, then:
migrate-loop --spec mymigration.md --resume
```

`--resume` reads `STATE.md` from the worktree and re-enters whatever phase
was interrupted. Stagnation counters are preserved on resume so a
fundamentally-stuck migration re-escalates fast rather than burning the new
budget.

## Common flags

| Flag | Default | Meaning |
|---|---|---|
| `--spec PATH` | (required) | Path to the migration spec markdown |
| `--repo PATH` | `.` | Source git repo (must contain `main`) |
| `--worktree-dir PATH` | `<repo>-migrate-<slug>` | Where to create the worktree |
| `--budget N` | `50` | Max CODE iterations |
| `--coverage-budget N` | `ceil(0.3 * --budget)` | Max COVER iterations |
| `--iter-timeout DUR` | `10m` | Per-iteration `claude -p` timeout |
| `--resume` | `false` | Resume from existing `STATE.md` |

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Migration complete; PR opened (or push-only if `gh` is missing) |
| `1` | Infrastructure error (git, claude binary, parser, etc.) |
| `2` | Migration escalation — see `ESCALATION.md` in the worktree |
| `130` | Interrupted (Ctrl-C). Resumable via `--resume`. |

## Escalation kinds

`ESCALATION.md`'s frontmatter records one of:

- `tests_vacuous` — planner wrote 0 failing tests; spec or planner output needs revision.
- `budget_exhausted` — main loop hit `--budget` without reaching green.
- `stagnation_after_redirect` — coder used the one-shot REDIRECT and still made no progress.
- `tests_seem_wrong` — coder wrote an objection to `FROZEN_TESTS.md`; review and decide.
- `iteration_timeout` — `claude -p` timed out 3 times in a row.

## Test freeze and `FROZEN_TESTS.md`

A pre-commit hook in the worktree rejects any commit touching `*_test.go`
files unless `ALLOW_TEST_EDITS=1` is set in the environment (which only the
PLAN and COVER phases set). The coder phase **cannot** modify tests.

If the coder genuinely believes a test is wrong, the only sanctioned channel
is to write its objection to `FROZEN_TESTS.md`. The harness reads that file
after each iteration; non-empty content triggers the `tests_seem_wrong`
escalation so a human can review.

## Layout of a completed run

```
<worktree>/
  STATE.md                    # current loop state (frontmatter YAML + readable body)
  FROZEN_TESTS.md             # empty unless coder objected
  ESCALATION.md               # only present if exit 2
  .migrate-loop.lock          # flock file held during run
  .git/hooks/pre-commit       # test-freeze enforcement
  .git/hooks/commit-msg       # commit-prefix enforcement
  ...migrated source...
```

The commit graph is the audit trail:

```
chore(migrate-loop): init <slug>
test(plan): <slug> failing test suite
wip(coder-1): ...
wip(coder-2): ...
test(coverage): close gap on <symbol>
chore(migrate-loop): completed <slug> in N iters
```
