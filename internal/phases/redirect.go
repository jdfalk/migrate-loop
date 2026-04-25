package phases

import (
	"context"
	"fmt"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/prompts"
	"github.com/jdfalk/migrate-loop/internal/state"
)

// Redirect runs the one-shot REDIRECT phase: instructs the agent to try a
// fundamentally different approach, then resets the stagnation streak and
// marks RedirectUsed so it cannot fire again. Always transitions back to CODE.
func Redirect(ctx context.Context, st *state.State, d Deps) error {
	prompt, err := prompts.RenderRedirect(prompts.RedirectInput{
		Iteration:        st.Iteration,
		OscillationLog:   summarizeOscillation(st.OscillationLog),
		WhatKeepsFailing: summarizeFailing(st.LastFailing),
		LastDiffs:        st.LastDiffSummary,
	})
	if err != nil {
		return err
	}
	resp, err := d.Agent.Run(ctx, agent.Request{
		Phase: agent.PhaseRedirect,
		Cwd:   d.Worktree.Path,
		AllowedTools: []string{
			"Read", "Edit", "Glob", "Grep",
			"Bash(go test:*)", "Bash(go vet:*)", "Bash(git add:*)", "Bash(git commit:*)",
		},
		Env: map[string]string{
			"EXPECTED_COMMIT_PREFIX": fmt.Sprintf("wip(coder-%d)", st.Iteration),
		},
		Prompt: prompt,
	})
	if err != nil {
		return fmt.Errorf("redirect: agent: %w", err)
	}
	st.TotalCostUSD += resp.Cost
	st.RedirectUsed = true
	st.StagnationStreak = 0
	st.Phase = state.PhaseCode
	return nil
}
