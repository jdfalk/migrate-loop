package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const preCommitScript = `#!/usr/bin/env bash
set -euo pipefail

# migrate-loop pre-commit hook
# Rejects changes to *_test.go unless ALLOW_TEST_EDITS=1.
# The agent's only sanctioned way to flag a test as wrong is to write to FROZEN_TESTS.md.

staged=$(git diff --cached --name-only --diff-filter=ACMR)

if [[ "${ALLOW_TEST_EDITS:-0}" != "1" ]]; then
  bad=$(echo "$staged" | grep -E '_test\.go$' || true)
  if [[ -n "$bad" ]]; then
    echo "migrate-loop pre-commit: test files are FROZEN during this phase." >&2
    echo "Files: $bad" >&2
    echo "If you believe a test is wrong, write your objection to FROZEN_TESTS.md and commit only that file." >&2
    exit 1
  fi
fi

exit 0
`

const commitMsgScript = `#!/usr/bin/env bash
set -euo pipefail

# migrate-loop commit-msg hook
# Rejects commit messages whose first line does not start with $EXPECTED_COMMIT_PREFIX (when set).

if [[ -z "${EXPECTED_COMMIT_PREFIX:-}" ]]; then
  exit 0
fi

msg_file="$1"
msg=$(head -n1 "$msg_file")

case "$msg" in
  "$EXPECTED_COMMIT_PREFIX"*) exit 0 ;;
  *)
    echo "migrate-loop commit-msg: commit message must start with '$EXPECTED_COMMIT_PREFIX'" >&2
    echo "Got: $msg" >&2
    exit 1
    ;;
esac
`

// HooksDirName is the directory inside the migrate-loop worktree where its
// hooks live. Pointed at by per-worktree core.hooksPath so the hooks fire
// only for commits made inside this worktree, not sibling worktrees of the
// same parent repo.
const HooksDirName = ".migrate-loop-hooks"

// InstallHook writes the migrate-loop hooks into a per-worktree hooks
// directory and configures this worktree (only) to use them via
// core.hooksPath. Sibling worktrees of the same parent repo are unaffected:
// their commits continue to use the shared (or unset) hooks path.
//
// Implementation notes:
//
//   - Sets extensions.worktreeConfig=true in the common config (idempotent).
//     Required for `git config --worktree` to take effect. Does not change
//     behavior for any worktree that doesn't set per-worktree config.
//   - Sets core.hooksPath ONLY in this worktree's per-worktree config, so
//     sibling worktrees keep their default hooks path (typically unset).
//
// Also creates an empty FROZEN_TESTS.md at the worktree root if missing —
// the sanctioned channel for the coder agent to flag a frozen test as wrong.
func (w *Worktree) InstallHook() error {
	ctx := context.Background()
	// 1. Enable per-worktree config (idempotent; required for --worktree).
	if err := exec.CommandContext(ctx, "git", "-C", w.Path, "config",
		"extensions.worktreeConfig", "true").Run(); err != nil {
		return fmt.Errorf("enable extensions.worktreeConfig: %w", err)
	}
	// 2. Create hooks directory inside the worktree itself.
	hooksDir := filepath.Join(w.Path, HooksDirName)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"),
		[]byte(preCommitScript), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "commit-msg"),
		[]byte(commitMsgScript), 0o755); err != nil {
		return err
	}
	// 3. Point ONLY this worktree at our hooks via per-worktree config.
	if err := exec.CommandContext(ctx, "git", "-C", w.Path, "config",
		"--worktree", "core.hooksPath", hooksDir).Run(); err != nil {
		return fmt.Errorf("set per-worktree core.hooksPath: %w", err)
	}
	// 4. Create FROZEN_TESTS.md if missing.
	frozen := filepath.Join(w.Path, "FROZEN_TESTS.md")
	if _, err := os.Stat(frozen); os.IsNotExist(err) {
		_ = os.WriteFile(frozen, []byte(""), 0o644)
	}
	return nil
}
