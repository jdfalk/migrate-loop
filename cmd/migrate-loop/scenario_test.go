package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/state"
)

// TestRun_BudgetExhausted: planner writes failing tests, coder makes no real
// progress for `Budget` iterations → budget_exhausted escalation, exit 2.
func TestRun_BudgetExhausted(t *testing.T) {
	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.md")
	mustWriteSpec(t, specPath, "demo-budget")
	srcRepo := setupSrcRepo(t, tmp)

	step := 0
	fa := &agent.FakeAgent{
		// Budget=2: 1 PLAN + 2 CODE iterations all making no progress.
		Responses: []agent.Response{
			{ExitCode: 0}, {ExitCode: 0}, {ExitCode: 0},
		},
		Editor: func(req agent.Request) error {
			step++
			switch step {
			case 1: // PLAN
				writeFile(filepath.Join(req.Cwd, "go.mod"), "module x\ngo 1.22\n")
				writeFile(filepath.Join(req.Cwd, "x.go"), "package x\nfunc Add(a, b int) int { return 0 }\n")
				writeFile(filepath.Join(req.Cwd, "x_test.go"),
					"package x\nimport \"testing\"\nfunc TestAdd(t *testing.T){ if Add(1,2)!=3 { t.Fail() } }\n")
				return commitAll(req.Cwd, "test(plan): demo-budget",
					"ALLOW_TEST_EDITS=1", "EXPECTED_COMMIT_PREFIX=test(plan)")
			default: // CODE no-op iterations
				writeFile(filepath.Join(req.Cwd, "noise.txt"), strings.Repeat("x", step))
				prefix := "wip(coder-" + itoa(step-1) + ")"
				return commitAll(req.Cwd, prefix+": noise", "EXPECTED_COMMIT_PREFIX="+prefix)
			}
		},
	}

	cfg := Config{
		SpecPath:       specPath,
		SourceRepo:     srcRepo,
		WorktreeDir:    filepath.Join(tmp, "wt"),
		Budget:         2,
		CoverageBudget: 1,
		IterTimeout:    10 * time.Second,
		Agent:          fa,
		SkipPR:         true,
	}
	err := Run(context.Background(), cfg)
	kind, ok := IsEscalation(err)
	if !ok {
		t.Fatalf("expected escalation, got %v", err)
	}
	if string(kind) != "budget_exhausted" {
		t.Errorf("kind = %q, want budget_exhausted", kind)
	}
	if _, err := os.Stat(filepath.Join(cfg.WorktreeDir, "ESCALATION.md")); err != nil {
		t.Errorf("ESCALATION.md missing: %v", err)
	}
}

// TestRun_Resume: simulate a prior escalation by hand-crafting a worktree and
// STATE.md, then re-invoke with --resume and verify the loop resumes from CODE.
func TestRun_Resume(t *testing.T) {
	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.md")
	mustWriteSpec(t, specPath, "demo-resume")
	srcRepo := setupSrcRepo(t, tmp)

	// First invocation: budget=1 will exhaust after one CODE iter.
	wtDir := filepath.Join(tmp, "wt")
	step := 0
	first := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}, {ExitCode: 0}},
		Editor: func(req agent.Request) error {
			step++
			switch step {
			case 1:
				writeFile(filepath.Join(req.Cwd, "go.mod"), "module x\ngo 1.22\n")
				writeFile(filepath.Join(req.Cwd, "x.go"), "package x\nfunc Add(a, b int) int { return 0 }\n")
				writeFile(filepath.Join(req.Cwd, "x_test.go"),
					"package x\nimport \"testing\"\nfunc TestAdd(t *testing.T){ if Add(1,2)!=3 { t.Fail() } }\n")
				return commitAll(req.Cwd, "test(plan): demo-resume",
					"ALLOW_TEST_EDITS=1", "EXPECTED_COMMIT_PREFIX=test(plan)")
			default:
				writeFile(filepath.Join(req.Cwd, "noise.txt"), "first")
				return commitAll(req.Cwd, "wip(coder-1): noise", "EXPECTED_COMMIT_PREFIX=wip(coder-1)")
			}
		},
	}
	cfg := Config{
		SpecPath: specPath, SourceRepo: srcRepo, WorktreeDir: wtDir,
		Budget: 1, CoverageBudget: 1, IterTimeout: 10 * time.Second,
		Agent: first, SkipPR: true,
	}
	err := Run(context.Background(), cfg)
	if _, ok := IsEscalation(err); !ok {
		t.Fatalf("expected first run to escalate, got %v", err)
	}

	// Resume: rewind STATE.md to CODE (simulating human fix), bump budget,
	// and re-run with an agent that fixes Add. Expect success.
	st, err := state.Read(filepath.Join(wtDir, "STATE.md"))
	if err != nil {
		t.Fatal(err)
	}
	st.Phase = state.PhaseCode
	st.EscalationReason = ""
	st.StagnationStreak = 0
	st.Budget = 5 // give it more room
	if err := state.Write(filepath.Join(wtDir, "STATE.md"), st); err != nil {
		t.Fatal(err)
	}

	// Second invocation: agent fixes the code on first iteration.
	step2 := 0
	second := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}, {ExitCode: 0}},
		Editor: func(req agent.Request) error {
			step2++
			if step2 == 1 {
				writeFile(filepath.Join(req.Cwd, "x.go"), "package x\nfunc Add(a, b int) int { return a + b }\n")
				prefix := "wip(coder-" + itoa(st.Iteration) + ")"
				return commitAll(req.Cwd, prefix+": fix Add", "EXPECTED_COMMIT_PREFIX="+prefix)
			}
			return nil
		},
	}
	cfg.Resume = true
	cfg.Agent = second
	cfg.Budget = 5
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatalf("resume run failed: %v", err)
	}
}

func mustWriteSpec(t *testing.T, path, slug string) {
	t.Helper()
	body := "---\n" +
		"title: " + slug + "\n" +
		"slug: " + slug + "\n" +
		"target_packages: [\".\"]\n" +
		"test_runner: \"go test -json ./...\"\n" +
		"prior_examples: []\n" +
		"success_criteria: [\"all tests pass\"]\n" +
		"---\n# " + slug + "\nbody\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupSrcRepo(t *testing.T, tmp string) string {
	t.Helper()
	srcRepo := filepath.Join(tmp, "src")
	mustGit(t, tmp, "init", srcRepo)
	mustGit(t, srcRepo, "config", "user.email", "t@t")
	mustGit(t, srcRepo, "config", "user.name", "t")
	mustGit(t, srcRepo, "commit", "--allow-empty", "-m", "init")
	mustGit(t, srcRepo, "branch", "-M", "main")
	return srcRepo
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	s := string(buf[i:])
	if neg {
		return "-" + s
	}
	return s
}
