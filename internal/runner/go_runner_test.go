package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGoRunner_Run(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not on PATH")
	}
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/m\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "m.go"), "package m\nfunc Add(a, b int) int { return a + b }\n")
	mustWrite(t, filepath.Join(dir, "m_test.go"), `package m
import "testing"
func TestPasses(t *testing.T) { if Add(1,2)!=3 { t.Fail() } }
func TestFails(t *testing.T)  { if Add(1,1)!=3 { t.Fail() } }
`)

	r := NewGoRunner("go test -race -json ./...")
	res, err := r.Run(context.Background(), dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Failing) != 1 || res.Failing[0].Test != "TestFails" {
		t.Errorf("Failing = %+v, want [TestFails]", res.Failing)
	}
	if len(res.Passing) != 1 || res.Passing[0].Test != "TestPasses" {
		t.Errorf("Passing = %+v", res.Passing)
	}
}

func TestGoRunner_CoverProfile(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not on PATH")
	}
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/c\ngo 1.22\n")
	mustWrite(t, filepath.Join(dir, "c.go"), "package c\nfunc Used() int { return 1 }\nfunc Unused() int { return 2 }\n")
	mustWrite(t, filepath.Join(dir, "c_test.go"), `package c
import "testing"
func TestUsed(t *testing.T) { if Used() != 1 { t.Fail() } }
`)
	r := NewGoRunner("")
	rep, err := r.CoverProfile(context.Background(), dir)
	if err != nil {
		t.Fatalf("CoverProfile: %v", err)
	}
	if len(rep.ByFile) == 0 {
		t.Fatalf("no files in coverage report")
	}
	gotUncovered := false
	for _, fc := range rep.ByFile {
		if len(fc.UncoveredLines) > 0 {
			gotUncovered = true
		}
	}
	if !gotUncovered {
		t.Error("expected at least one uncovered line in Unused() function")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
