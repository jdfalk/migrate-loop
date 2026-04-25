package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCreate_BranchFromMain(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	srcRepo := filepath.Join(root, "src")
	mustGit(t, root, "init", srcRepo)
	mustGit(t, srcRepo, "config", "user.email", "t@t")
	mustGit(t, srcRepo, "config", "user.name", "t")
	mustGit(t, srcRepo, "commit", "--allow-empty", "-m", "init")
	mustGit(t, srcRepo, "branch", "-M", "main")

	wtPath := filepath.Join(root, "wt")
	wt, err := Create(context.Background(), Options{
		SourceRepo:  srcRepo,
		BranchName:  "migrate/demo",
		WorktreeDir: wtPath,
		BaseRef:     "main",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if wt.Path != wtPath {
		t.Errorf("Path = %q", wt.Path)
	}
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
		t.Errorf("worktree .git missing: %v", err)
	}
}

func TestCommitAndDiffSummary(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	mustGit(t, root, "init", src)
	mustGit(t, src, "config", "user.email", "t@t")
	mustGit(t, src, "config", "user.name", "t")
	mustGit(t, src, "commit", "--allow-empty", "-m", "init")
	mustGit(t, src, "branch", "-M", "main")

	wtPath := filepath.Join(root, "wt")
	wt, err := Create(context.Background(), Options{SourceRepo: src, BranchName: "feat/x", WorktreeDir: wtPath, BaseRef: "main"})
	if err != nil {
		t.Fatal(err)
	}
	mustGit(t, wt.Path, "config", "user.email", "t@t")
	mustGit(t, wt.Path, "config", "user.name", "t")

	if err := os.WriteFile(filepath.Join(wt.Path, "f.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := wt.Commit(context.Background(), "feat: add f", nil); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	out, err := wt.DiffSummary(context.Background(), "main..HEAD")
	if err != nil {
		t.Fatalf("DiffSummary: %v", err)
	}
	if out == "" {
		t.Error("DiffSummary should be non-empty after a commit")
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
