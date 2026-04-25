package worktree

import (
	"path/filepath"
	"testing"
)

func TestLock_Exclusive(t *testing.T) {
	dir := t.TempDir()
	l1, err := Lock(filepath.Join(dir, ".migrate-loop.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Release()

	if _, err := Lock(filepath.Join(dir, ".migrate-loop.lock")); err == nil {
		t.Error("expected second Lock to fail")
	}
}

func TestLock_ReleaseAllowsRelock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lock")
	l, err := Lock(p)
	if err != nil {
		t.Fatal(err)
	}
	l.Release()
	l2, err := Lock(p)
	if err != nil {
		t.Fatalf("relock after release: %v", err)
	}
	l2.Release()
}
