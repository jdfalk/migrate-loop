package phases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/prompts"
	"github.com/jdfalk/migrate-loop/internal/state"
)

// Code runs ONE coder iteration. Returns advance=true if all green and the
// state was advanced to COVER. The caller loops until advance OR
// budget exhausted OR escalation.
func Code(ctx context.Context, st *state.State, d Deps) (bool, error) {
	failing := summarizeFailing(st.LastFailing)
	tail := tailLines(st.LastDiffSummary, 40)
	osc := ""
	if len(st.OscillationLog) > 0 {
		osc = "Earlier oscillations: " + summarizeOscillation(st.OscillationLog)
	}
	prompt, err := prompts.RenderCoder(prompts.CoderInput{
		Iteration:       st.Iteration,
		FailingTests:    failing,
		LastTestOutput:  tail,
		LastDiff:        st.LastDiffSummary,
		OscillationNote: osc,
	})
	if err != nil {
		return false, err
	}
	resp, err := d.Agent.Run(ctx, agent.Request{
		Phase: agent.PhaseCode,
		Cwd:   d.Worktree.Path,
		AllowedTools: []string{
			"Read", "Edit", "Glob", "Grep",
			"Bash(go test:*)", "Bash(go vet:*)", "Bash(go build:*)",
			"Bash(git add:*)", "Bash(git commit:*)",
		},
		Env: map[string]string{
			"EXPECTED_COMMIT_PREFIX": fmt.Sprintf("wip(coder-%d)", st.Iteration),
		},
		Prompt: prompt,
	})
	if err != nil {
		return false, fmt.Errorf("code: agent: %w", err)
	}
	st.TotalCostUSD += resp.Cost
	st.BudgetUsed++

	if frozen := readFrozenObjection(d.Worktree.Path); frozen != "" {
		return false, fmt.Errorf("tests_seem_wrong: %s", frozen)
	}

	res, err := d.Runner.Run(ctx, d.Worktree.Path)
	if err != nil {
		return false, fmt.Errorf("code: runner: %w", err)
	}
	prog := state.DetectProgress(st.LastFailing, res.Failing)
	st.LastFailing = res.Failing

	if prog.AllGreen {
		st.Phase = state.PhaseCover
		st.StagnationStreak = 0
		return true, nil
	}
	if prog.IsProgress {
		st.StagnationStreak = 0
		if prog.Oscillation {
			st.OscillationLog = append(st.OscillationLog, state.OscillationEvent{
				Iteration: st.Iteration,
				Note:      fmt.Sprintf("rotated: now-passing=%d newly-failing=%d", len(prog.NowPassing), len(prog.NewlyFailing)),
			})
		}
	} else {
		st.StagnationStreak++
	}
	st.Iteration++

	if st.StagnationStreak >= 3 && !st.RedirectUsed {
		st.Phase = state.PhaseRedirect
	} else if st.StagnationStreak >= 4 && st.RedirectUsed {
		return false, errors.New("stagnation_after_redirect")
	}
	if st.BudgetUsed >= st.Budget && st.Phase != state.PhaseCover {
		return false, errors.New("budget_exhausted")
	}
	return false, nil
}

func summarizeFailing(ids []state.TestID) string {
	if len(ids) == 0 {
		return "(none)"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.Package + "::" + id.Test
	}
	return strings.Join(parts, ", ")
}

func summarizeOscillation(ev []state.OscillationEvent) string {
	parts := make([]string, len(ev))
	for i, e := range ev {
		parts[i] = fmt.Sprintf("iter%d:%s", e.Iteration, e.Note)
	}
	return strings.Join(parts, "; ")
}

func tailLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func readFrozenObjection(cwd string) string {
	body, err := os.ReadFile(filepath.Join(cwd, "FROZEN_TESTS.md"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}
