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
