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

func TestRedirect_ResetsStreakAndMarksUsed(t *testing.T) {
	wt := setupWorktree(t)
	seedRedRepo(t, wt.Path)
	if err := wt.Commit(context.Background(), "test(plan): demo", map[string]string{"ALLOW_TEST_EDITS": "1", "EXPECTED_COMMIT_PREFIX": "test(plan)"}); err != nil {
		t.Fatal(err)
	}
	st := &state.State{Slug: "demo", Phase: state.PhaseRedirect, Iteration: 4, Budget: 50, StagnationStreak: 3}
	fa := &agent.FakeAgent{
		Responses: []agent.Response{{ExitCode: 0}},
		Editor: func(req agent.Request) error {
			os.WriteFile(filepath.Join(req.Cwd, "x.go"), []byte("package x\nfunc Add(a,b int) int { return a+b }\n"), 0o644)
			return wt.Commit(context.Background(), "wip(coder-4): different shape", map[string]string{"EXPECTED_COMMIT_PREFIX": "wip(coder-4)"})
		},
	}
	deps := Deps{Agent: fa, Runner: runner.NewGoRunner(""), Worktree: wt}
	if err := Redirect(context.Background(), st, deps); err != nil {
		t.Fatal(err)
	}
	if !st.RedirectUsed {
		t.Error("RedirectUsed should be true")
	}
	if st.StagnationStreak != 0 {
		t.Errorf("StagnationStreak = %d, want 0", st.StagnationStreak)
	}
	if st.Phase != state.PhaseCode {
		t.Errorf("phase after Redirect = %v, want CODE", st.Phase)
	}
}
