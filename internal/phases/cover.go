package phases

import (
	"context"
	"fmt"
	"strings"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/prompts"
	"github.com/jdfalk/migrate-loop/internal/runner"
	"github.com/jdfalk/migrate-loop/internal/state"
)

func Cover(ctx context.Context, st *state.State, d Deps) error {
	rep, err := d.Runner.CoverProfile(ctx, d.Worktree.Path)
	if err != nil {
		return fmt.Errorf("cover: profile: %w", err)
	}
	gaps := summarizeGaps(rep)
	if gaps == "" {
		st.Phase = state.PhasePR
		return nil
	}
	if st.CoverageBudgetUsed >= st.CoverageBudget {
		st.Phase = state.PhasePR
		return nil
	}
	prompt, err := prompts.RenderCover(prompts.CoverInput{UncoveredGaps: gaps})
	if err != nil {
		return err
	}
	resp, err := d.Agent.Run(ctx, agent.Request{
		Phase: agent.PhaseCover,
		Cwd:   d.Worktree.Path,
		AllowedTools: []string{
			"Read", "Write", "Edit", "Glob", "Grep",
			"Bash(go test:*)", "Bash(git add:*)", "Bash(git commit:*)",
		},
		Env: map[string]string{
			"ALLOW_TEST_EDITS":       "1",
			"EXPECTED_COMMIT_PREFIX": "test(coverage)",
		},
		Prompt: prompt,
	})
	if err != nil {
		return fmt.Errorf("cover: agent: %w", err)
	}
	st.TotalCostUSD += resp.Cost
	st.CoverageBudgetUsed++

	res, err := d.Runner.Run(ctx, d.Worktree.Path)
	if err != nil {
		return err
	}
	if len(res.Failing) > 0 {
		// New tests are red; re-enter LOOP via main state machine
		st.Phase = state.PhaseCode
		st.LastFailing = res.Failing
		return nil
	}
	st.Phase = state.PhasePR
	return nil
}

func summarizeGaps(rep runner.CoverageReport) string {
	parts := []string{}
	for f, fc := range rep.ByFile {
		if len(fc.UncoveredLines) == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("- %s: lines %v", f, fc.UncoveredLines))
	}
	return strings.Join(parts, "\n")
}
