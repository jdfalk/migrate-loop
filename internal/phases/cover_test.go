package phases

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/runner"
	"github.com/jdfalk/migrate-loop/internal/state"
)

func TestCover_FullyCoveredAdvancesToPR(t *testing.T) {
	wt := setupWorktree(t)
	// fully-covered tiny repo
	os.WriteFile(filepath.Join(wt.Path, "go.mod"), []byte("module x\ngo 1.22\n"), 0o644)
	os.WriteFile(filepath.Join(wt.Path, "x.go"), []byte("package x\nfunc Add(a,b int) int { return a+b }\n"), 0o644)
	os.WriteFile(filepath.Join(wt.Path, "x_test.go"), []byte("package x\nimport \"testing\"\nfunc TestAdd(t *testing.T){ if Add(1,2)!=3 { t.Fail() } }\n"), 0o644)
	wt.Commit(context.Background(), "test(plan): demo", map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"})

	st := &state.State{Slug: "demo", Phase: state.PhaseCover, CoverageBudget: 5}
	fa := &agent.FakeAgent{Responses: []agent.Response{{ExitCode: 0}}}
	deps := Deps{Agent: fa, Runner: runner.NewGoRunner(""), Worktree: wt}
	if err := Cover(context.Background(), st, deps); err != nil {
		t.Fatal(err)
	}
	if st.Phase != state.PhasePR {
		t.Errorf("phase = %v, want PR (no gaps to fill)", st.Phase)
	}
}
