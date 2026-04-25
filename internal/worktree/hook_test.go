package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHookRejectsTestEditWithoutEnv(t *testing.T) {
	wt := setupWorktreeWithHook(t)

	mustWrite(t, filepath.Join(wt.Path, "x_test.go"), "package x\n")
	mustGit(t, wt.Path, "add", "x_test.go")
	c := exec.Command("git", "-C", wt.Path, "commit", "-m", "wip(coder-1): try thing")
	out, err := c.CombinedOutput()
	if err == nil {
		t.Fatalf("commit should have been rejected, got success:\n%s", out)
	}
}

func TestHookAllowsTestEditWithEnv(t *testing.T) {
	wt := setupWorktreeWithHook(t)

	mustWrite(t, filepath.Join(wt.Path, "x_test.go"), "package x\n")
	mustGit(t, wt.Path, "add", "x_test.go")
	c := exec.Command("git", "-C", wt.Path, "commit", "-m", "test(plan): add red")
	c.Env = append(os.Environ(), "ALLOW_TEST_EDITS=1", "EXPECTED_COMMIT_PREFIX=test(plan)")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("commit should be allowed:\n%s\n%v", out, err)
	}
}

func TestHookRejectsWrongCommitPrefix(t *testing.T) {
	wt := setupWorktreeWithHook(t)
	mustWrite(t, filepath.Join(wt.Path, "x.go"), "package x\n")
	mustGit(t, wt.Path, "add", "x.go")
	c := exec.Command("git", "-C", wt.Path, "commit", "-m", "feat: wrong prefix")
	c.Env = append(os.Environ(), "EXPECTED_COMMIT_PREFIX=wip(coder-1)")
	out, err := c.CombinedOutput()
	if err == nil {
		t.Fatalf("commit should have been rejected:\n%s", out)
	}
}

// TestHook_DoesNotAffectSiblingWorktree is the regression guard for the
// cross-contamination bug: when migrate-loop installed hooks into the
// common gitdir, sibling worktrees of the same parent repo would get
// their commits rejected by the test-freeze hook even though they had
// nothing to do with the migration.
//
// With per-worktree core.hooksPath, the migrate-loop hooks fire only for
// commits made inside the migrate-loop worktree. A sibling worktree of
// the same parent repo can commit *_test.go files freely.
func TestHook_DoesNotAffectSiblingWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	mustGit(t, root, "init", src)
	mustGit(t, src, "config", "user.email", "test@test")
	mustGit(t, src, "config", "user.name", "test")
	mustGit(t, src, "commit", "--allow-empty", "-m", "init")
	mustGit(t, src, "branch", "-M", "main")

	// Create the migrate-loop worktree and install its hooks.
	mlPath := filepath.Join(root, "ml-wt")
	mlWT, err := Create(context.Background(), Options{
		SourceRepo: src, WorktreeDir: mlPath, BranchName: "migrate/demo", BaseRef: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustGit(t, mlWT.Path, "config", "user.email", "test@test")
	mustGit(t, mlWT.Path, "config", "user.name", "test")
	if err := mlWT.InstallHook(); err != nil {
		t.Fatal(err)
	}

	// Create a SIBLING worktree (simulates user's other in-flight work).
	siblingPath := filepath.Join(root, "sibling-wt")
	mustGit(t, src, "worktree", "add", siblingPath, "-b", "sibling-feature")
	mustGit(t, siblingPath, "config", "user.email", "test@test")
	mustGit(t, siblingPath, "config", "user.name", "test")

	// Try to commit a test file in the sibling — should SUCCEED (no
	// migrate-loop hook applies). With the buggy common-dir install this
	// would fail.
	mustWrite(t, filepath.Join(siblingPath, "y_test.go"), "package y\n")
	mustGit(t, siblingPath, "add", "y_test.go")
	out, err := exec.Command("git", "-C", siblingPath, "commit", "-m", "feat: add test").CombinedOutput()
	if err != nil {
		t.Fatalf("sibling worktree commit should succeed (migrate-loop hooks must NOT leak):\n%s\nerr: %v", out, err)
	}

	// And the migrate-loop worktree's hook still fires for its own commits.
	mustWrite(t, filepath.Join(mlWT.Path, "x_test.go"), "package x\n")
	mustGit(t, mlWT.Path, "add", "x_test.go")
	out, err = exec.Command("git", "-C", mlWT.Path, "commit", "-m", "wip(coder-1): try thing").CombinedOutput()
	if err == nil {
		t.Fatalf("migrate-loop worktree commit should still be rejected by hook:\n%s", out)
	}
}

func setupWorktreeWithHook(t *testing.T) *Worktree {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	mustGit(t, root, "init", src)
	mustGit(t, src, "config", "user.email", "test@test")
	mustGit(t, src, "config", "user.name", "test")
	mustGit(t, src, "commit", "--allow-empty", "-m", "init")
	mustGit(t, src, "branch", "-M", "main")
	wtPath := filepath.Join(root, "wt")
	wt, err := Create(context.Background(), Options{SourceRepo: src, WorktreeDir: wtPath, BranchName: "feat/x", BaseRef: "main"})
	if err != nil {
		t.Fatal(err)
	}
	mustGit(t, wt.Path, "config", "user.email", "test@test")
	mustGit(t, wt.Path, "config", "user.name", "test")
	if err := wt.InstallHook(); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(wt.Path, "FROZEN_TESTS.md"), "")
	return wt
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
