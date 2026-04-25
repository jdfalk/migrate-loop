package phases

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/runner"
	"github.com/jdfalk/migrate-loop/internal/spec"
	"github.com/jdfalk/migrate-loop/internal/state"
)

func TestPlan_HappyPath(t *testing.T) {
	wt := setupWorktree(t)
	sp := &spec.Spec{
		Slug: "demo", TargetPackages: []string{"."}, TestRunner: "go test -json ./...",
		Body: "add two ints",
	}
	st := &state.State{Slug: "demo", Phase: state.PhaseInit, Budget: 50}

	fa := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0, Cost: 0.10}},
		Editor: func(req agent.Request) error {
			if err := os.WriteFile(filepath.Join(req.Cwd, "go.mod"), []byte("module x\ngo 1.22\n"), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(req.Cwd, "x.go"), []byte("package x\nfunc Add(a,b int) int { return 0 }\n"), 0o644); err != nil {
				return err
			}
			testSrc := "package x\nimport \"testing\"\nfunc TestAdd(t *testing.T) { if Add(1,2)!=3 { t.Fail() } }\n"
			if err := os.WriteFile(filepath.Join(req.Cwd, "x_test.go"), []byte(testSrc), 0o644); err != nil {
				return err
			}
			env := map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"}
			return wt.Commit(context.Background(), "test(plan): demo failing test suite", env)
		},
	}

	deps := Deps{
		Agent:    fa,
		Runner:   runner.NewGoRunner("go test -json ./..."),
		Worktree: wt,
	}
	if err := Plan(context.Background(), st, sp, deps); err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if st.Phase != state.PhaseCode {
		t.Errorf("phase after PLAN = %v, want CODE", st.Phase)
	}
	if len(st.LastFailing) == 0 {
		t.Errorf("expected LastFailing populated after PLAN")
	}
}

func TestPlan_VacuousTestsEscalate(t *testing.T) {
	wt := setupWorktree(t)
	sp := &spec.Spec{Slug: "demo", TargetPackages: []string{"."}, TestRunner: "go test -json ./...", Body: "x"}
	st := &state.State{Slug: "demo", Phase: state.PhaseInit}

	fa := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}},
		Editor: func(req agent.Request) error {
			os.WriteFile(filepath.Join(req.Cwd, "go.mod"), []byte("module x\ngo 1.22\n"), 0o644)
			os.WriteFile(filepath.Join(req.Cwd, "x.go"), []byte("package x\n"), 0o644)
			testSrc := "package x\nimport \"testing\"\nfunc TestPasses(t *testing.T){}\n"
			os.WriteFile(filepath.Join(req.Cwd, "x_test.go"), []byte(testSrc), 0o644)
			env := map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"}
			return wt.Commit(context.Background(), "test(plan): vacuous", env)
		},
	}
	deps := Deps{Agent: fa, Runner: runner.NewGoRunner(""), Worktree: wt}
	err := Plan(context.Background(), st, sp, deps)
	if err == nil || !strings.Contains(err.Error(), "tests_vacuous") {
		t.Fatalf("expected tests_vacuous error, got %v", err)
	}
}
