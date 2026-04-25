package phases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/prompts"
	"github.com/jdfalk/migrate-loop/internal/spec"
	"github.com/jdfalk/migrate-loop/internal/state"
)

func Plan(ctx context.Context, st *state.State, sp *spec.Spec, d Deps) error {
	priors, err := loadPriors(sp)
	if err != nil {
		return fmt.Errorf("plan: load priors: %w", err)
	}
	prompt, err := prompts.RenderPlanner(prompts.PlannerInput{
		Slug:           sp.Slug,
		SpecBody:       sp.Body,
		PriorExamples:  priors,
		TargetPackages: sp.TargetPackages,
		TestRunner:     sp.TestRunner,
	})
	if err != nil {
		return err
	}
	resp, err := d.Agent.Run(ctx, agent.Request{
		Phase: agent.PhasePlan,
		Cwd:   d.Worktree.Path,
		AllowedTools: []string{
			"Read", "Write", "Edit", "Glob", "Grep",
			"Bash(go test:*)", "Bash(go build:*)", "Bash(git add:*)", "Bash(git commit:*)",
		},
		Env: map[string]string{
			"ALLOW_TEST_EDITS":       "1",
			"EXPECTED_COMMIT_PREFIX": "test(plan)",
		},
		Prompt: prompt,
	})
	if err != nil {
		return fmt.Errorf("plan: agent: %w", err)
	}
	st.TotalCostUSD += resp.Cost

	res, err := d.Runner.Run(ctx, d.Worktree.Path)
	if err != nil {
		return fmt.Errorf("plan: runner: %w", err)
	}
	if len(res.Failing) == 0 {
		return errors.New("tests_vacuous: planner finished but 0 tests fail")
	}
	st.LastFailing = res.Failing
	st.Phase = state.PhaseCode
	st.Iteration = 1
	return nil
}

func loadPriors(sp *spec.Spec) ([]prompts.PriorExample, error) {
	out := make([]prompts.PriorExample, 0, len(sp.PriorExamples))
	specDir := filepath.Dir(sp.FilePath)
	for _, ref := range sp.PriorExamples {
		path := ref
		if !filepath.IsAbs(path) {
			path = filepath.Join(specDir, path)
		}
		if _, err := os.Stat(path); err != nil {
			out = append(out, prompts.PriorExample{Path: ref, Content: "(prior not resolvable as file: " + ref + ")"})
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, prompts.PriorExample{Path: ref, Content: string(body)})
	}
	return out, nil
}
