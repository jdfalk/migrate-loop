package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jdfalk/migrate-loop/internal/agent"
)

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// scriptedHappyAgent returns a FakeAgent that simulates:
//
//	PLAN  → writes go.mod + impl stub + failing test, commits as test(plan)
//	CODE  → fixes Add to make TestAdd pass, commits as wip(coder-1)
//	COVER → no-op (coverage subagent finds nothing or just exits)
func scriptedHappyAgent(t *testing.T) *agent.FakeAgent {
	t.Helper()
	step := 0
	return &agent.FakeAgent{
		Responses: []agent.Response{
			{ExitCode: 0, Cost: 0.10},
			{ExitCode: 0, Cost: 0.20},
			{ExitCode: 0, Cost: 0.05},
		},
		Editor: func(req agent.Request) error {
			step++
			switch step {
			case 1: // PLAN
				if err := writeFile(filepath.Join(req.Cwd, "go.mod"), "module x\ngo 1.22\n"); err != nil {
					return err
				}
				if err := writeFile(filepath.Join(req.Cwd, "x.go"), "package x\nfunc Add(a, b int) int { return 0 }\n"); err != nil {
					return err
				}
				testSrc := "package x\nimport \"testing\"\nfunc TestAdd(t *testing.T){ if Add(1,2)!=3 { t.Fail() } }\n"
				if err := writeFile(filepath.Join(req.Cwd, "x_test.go"), testSrc); err != nil {
					return err
				}
				return commitAll(req.Cwd, "test(plan): trivial-add failing test suite",
					"ALLOW_TEST_EDITS=1", "EXPECTED_COMMIT_PREFIX=test(plan)")
			case 2: // CODE iter 1
				if err := writeFile(filepath.Join(req.Cwd, "x.go"), "package x\nfunc Add(a, b int) int { return a + b }\n"); err != nil {
					return err
				}
				return commitAll(req.Cwd, "wip(coder-1): fix Add", "EXPECTED_COMMIT_PREFIX=wip(coder-1)")
			case 3: // COVER (no-op — gaps may or may not exist)
				return nil
			}
			return nil
		},
	}
}

// vacuousAgent simulates a planner that writes only-passing tests, which
// should trigger the tests_vacuous escalation in PLAN.
func vacuousAgent(t *testing.T) *agent.FakeAgent {
	t.Helper()
	return &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}},
		Editor: func(req agent.Request) error {
			if err := writeFile(filepath.Join(req.Cwd, "go.mod"), "module x\ngo 1.22\n"); err != nil {
				return err
			}
			if err := writeFile(filepath.Join(req.Cwd, "x.go"), "package x\n"); err != nil {
				return err
			}
			testSrc := "package x\nimport \"testing\"\nfunc TestVacuous(t *testing.T){}\n"
			if err := writeFile(filepath.Join(req.Cwd, "x_test.go"), testSrc); err != nil {
				return err
			}
			return commitAll(req.Cwd, "test(plan): vacuous",
				"ALLOW_TEST_EDITS=1", "EXPECTED_COMMIT_PREFIX=test(plan)")
		},
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func commitAll(dir, msg string, env ...string) error {
	c1 := exec.Command("git", "add", "-A")
	c1.Dir = dir
	c1.Env = append(os.Environ(), env...)
	if out, err := c1.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}
	c2 := exec.Command("git", "commit", "-m", msg)
	c2.Dir = dir
	c2.Env = append(os.Environ(), env...)
	if out, err := c2.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	return nil
}
