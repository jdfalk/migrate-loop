package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// InstallHook writes the pre-commit hook into the worktree's gitdir and
// creates an empty FROZEN_TESTS.md if missing.
func (w *Worktree) InstallHook() error {
	gitDir, err := w.gitDir()
	if err != nil {
		return err
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(preCommitScript), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "commit-msg"), []byte(commitMsgScript), 0o755); err != nil {
		return err
	}
	frozen := filepath.Join(w.Path, "FROZEN_TESTS.md")
	if _, err := os.Stat(frozen); os.IsNotExist(err) {
		_ = os.WriteFile(frozen, []byte(""), 0o644)
	}
	return nil
}

func (w *Worktree) gitDir() (string, error) {
	out, err := exec.Command("git", "-C", w.Path, "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse: %w", err)
	}
	gd := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(w.Path, gd)
	}
	return gd, nil
}
