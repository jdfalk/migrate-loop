package phases

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/runner"
	"github.com/jdfalk/migrate-loop/internal/state"
)

func seedRedRepo(t *testing.T, wtPath string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(wtPath, "go.mod"), []byte("module x\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "x.go"), []byte("package x\nfunc Add(a,b int) int { return 0 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "x_test.go"), []byte("package x\nimport \"testing\"\nfunc TestAdd(t *testing.T){ if Add(1,2)!=3 { t.Fail() } }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCode_MakesProgressAndAdvancesAtGreen(t *testing.T) {
	wt := setupWorktree(t)
	seedRedRepo(t, wt.Path)
	if err := wt.Commit(context.Background(), "test(plan): demo", map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"}); err != nil {
		t.Fatal(err)
	}

	st := &state.State{
		Slug: "demo", Phase: state.PhaseCode, Iteration: 1, Budget: 5,
		LastFailing: []state.TestID{{Package: "x", Test: "TestAdd"}},
	}

	fa := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}},
		Editor: func(req agent.Request) error {
			if err := os.WriteFile(filepath.Join(req.Cwd, "x.go"), []byte("package x\nfunc Add(a,b int) int { return a+b }\n"), 0o644); err != nil {
				return err
			}
			return wt.Commit(context.Background(), "wip(coder-1): fix Add", map[string]string{"EXPECTED_COMMIT_PREFIX": "wip(coder-1)"})
		},
	}
	deps := Deps{Agent: fa, Runner: runner.NewGoRunner(""), Worktree: wt}

	advance, err := Code(context.Background(), st, deps)
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if !advance {
		t.Errorf("expected advance=true at green")
	}
	if st.Phase != state.PhaseCover {
		t.Errorf("phase after green CODE = %v, want COVER", st.Phase)
	}
}

func TestCode_StagnationIncrementsCounter(t *testing.T) {
	wt := setupWorktree(t)
	seedRedRepo(t, wt.Path)
	if err := wt.Commit(context.Background(), "test(plan): demo", map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"}); err != nil {
		t.Fatal(err)
	}

	st := &state.State{
		Slug: "demo", Phase: state.PhaseCode, Iteration: 1, Budget: 10,
		LastFailing: []state.TestID{{Package: "x", Test: "TestAdd"}},
	}
	fa := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}},
		Editor: func(req agent.Request) error {
			if err := os.WriteFile(filepath.Join(req.Cwd, "noise.txt"), []byte("noop"), 0o644); err != nil {
				return err
			}
			return wt.Commit(context.Background(), "wip(coder-1): no-op", map[string]string{"EXPECTED_COMMIT_PREFIX": "wip(coder-1)"})
		},
	}
	deps := Deps{Agent: fa, Runner: runner.NewGoRunner(""), Worktree: wt}
	_, err := Code(context.Background(), st, deps)
	if err != nil {
		t.Fatal(err)
	}
	if st.StagnationStreak != 1 {
		t.Errorf("StagnationStreak = %d, want 1", st.StagnationStreak)
	}
}

func TestCode_FrozenTestsEscalates(t *testing.T) {
	wt := setupWorktree(t)
	seedRedRepo(t, wt.Path)
	if err := wt.Commit(context.Background(), "test(plan): demo", map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"}); err != nil {
		t.Fatal(err)
	}

	st := &state.State{
		Slug: "demo", Phase: state.PhaseCode, Iteration: 1, Budget: 10,
		LastFailing: []state.TestID{{Package: "x", Test: "TestAdd"}},
	}
	fa := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}},
		Editor: func(req agent.Request) error {
			return os.WriteFile(filepath.Join(req.Cwd, "FROZEN_TESTS.md"), []byte("TestAdd's expected value seems wrong\n"), 0o644)
		},
	}
	deps := Deps{Agent: fa, Runner: runner.NewGoRunner(""), Worktree: wt}
	_, err := Code(context.Background(), st, deps)
	if err == nil || !strings.Contains(err.Error(), "tests_seem_wrong") {
		t.Fatalf("expected tests_seem_wrong, got %v", err)
	}
}
