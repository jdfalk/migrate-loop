package phases

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jdfalk/migrate-loop/internal/worktree"
)

func setupWorktree(t *testing.T) *worktree.Worktree {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	mustCmd(t, root, "git", "init", src)
	mustCmd(t, src, "git", "config", "user.email", "t@t")
	mustCmd(t, src, "git", "config", "user.name", "t")
	mustCmd(t, src, "git", "commit", "--allow-empty", "-m", "init")
	mustCmd(t, src, "git", "branch", "-M", "main")
	wtPath := filepath.Join(root, "wt")
	wt, err := worktree.Create(context.Background(), worktree.Options{
		SourceRepo: src, WorktreeDir: wtPath, BranchName: "migrate/demo", BaseRef: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustCmd(t, wt.Path, "git", "config", "user.email", "t@t")
	mustCmd(t, wt.Path, "git", "config", "user.name", "t")
	if err := wt.InstallHook(); err != nil {
		t.Fatal(err)
	}
	return wt
}

func mustCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	c := exec.Command(name, args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
