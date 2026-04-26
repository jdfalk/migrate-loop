package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jdfalk/migrate-loop/internal/agent"
	"github.com/jdfalk/migrate-loop/internal/escalate"
	"github.com/jdfalk/migrate-loop/internal/phases"
	"github.com/jdfalk/migrate-loop/internal/runner"
	"github.com/jdfalk/migrate-loop/internal/spec"
	"github.com/jdfalk/migrate-loop/internal/state"
	"github.com/jdfalk/migrate-loop/internal/worktree"
)

type Config struct {
	SpecPath       string
	SourceRepo     string
	WorktreeDir    string
	Budget         int
	CoverageBudget int
	IterTimeout    time.Duration
	Agent          agent.Agent // injected; defaults to CLIAgent in main()
	SkipPR         bool        // for tests
	Resume         bool
	Overwrite      bool // destroy existing worktree+branch and start fresh
}

func Run(ctx context.Context, cfg Config) error {
	sp, err := spec.ParseFile(cfg.SpecPath)
	if err != nil {
		return err
	}
	if cfg.WorktreeDir == "" {
		cfg.WorktreeDir = filepath.Join(filepath.Dir(cfg.SourceRepo), filepath.Base(cfg.SourceRepo)+"-migrate-"+sp.Slug)
	}
	if cfg.CoverageBudget == 0 {
		cfg.CoverageBudget = (cfg.Budget + 2) / 3 // ceil(0.3 * budget) approx
		if cfg.CoverageBudget < 1 {
			cfg.CoverageBudget = 1
		}
	}

	statePath := filepath.Join(cfg.WorktreeDir, "STATE.md")
	var wt *worktree.Worktree
	var st *state.State
	if cfg.Resume {
		st, err = state.Read(statePath)
		if err != nil {
			return fmt.Errorf("resume: %w", err)
		}
		wt = &worktree.Worktree{Path: cfg.WorktreeDir, BranchName: "migrate/" + sp.Slug}
	} else {
		if cfg.Overwrite {
			// Destroy the existing worktree and branch so Create starts clean.
			_ = worktree.Remove(ctx, cfg.SourceRepo, cfg.WorktreeDir, "migrate/"+sp.Slug)
		}
		baseRef := resolveBaseRef(ctx, cfg.SourceRepo)
		wt, err = worktree.Create(ctx, worktree.Options{
			SourceRepo:  cfg.SourceRepo,
			BranchName:  "migrate/" + sp.Slug,
			WorktreeDir: cfg.WorktreeDir,
			BaseRef:     baseRef,
		})
		if err != nil {
			return err
		}
		// Ensure committer identity exists in this worktree (covers fresh
		// repos without a global git config — required for hook-driven
		// commits to succeed).
		ensureGitIdentity(ctx, wt.Path)
		if err := wt.InstallHook(); err != nil {
			return err
		}
		st = &state.State{
			Slug:           sp.Slug,
			Phase:          state.PhaseInit,
			Budget:         cfg.Budget,
			CoverageBudget: cfg.CoverageBudget,
		}
		if err := state.Write(statePath, st); err != nil {
			return err
		}
		_ = wt.Commit(ctx, fmt.Sprintf("chore(migrate-loop): init %s", sp.Slug),
			map[string]string{"EXPECTED_COMMIT_PREFIX": "chore(migrate-loop)"})
	}

	lock, err := worktree.Lock(filepath.Join(cfg.WorktreeDir, ".migrate-loop.lock"))
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	defer lock.Release()

	deps := phases.Deps{
		Agent:    cfg.Agent,
		Runner:   runner.NewGoRunner(sp.TestRunner),
		Worktree: wt,
	}

	for {
		switch st.Phase {
		case state.PhaseInit:
			st.Phase = state.PhasePlan
		case state.PhasePlan:
			if err := phases.Plan(ctx, st, sp, deps); err != nil {
				return handleErr(err, st, wt, statePath)
			}
		case state.PhaseCode:
			if _, err := phases.Code(ctx, st, deps); err != nil {
				return handleErr(err, st, wt, statePath)
			}
		case state.PhaseRedirect:
			if err := phases.Redirect(ctx, st, deps); err != nil {
				return handleErr(err, st, wt, statePath)
			}
		case state.PhaseCover:
			if err := phases.Cover(ctx, st, deps); err != nil {
				return handleErr(err, st, wt, statePath)
			}
		case state.PhasePR:
			if !cfg.SkipPR {
				if err := phases.PR(ctx, st, deps); err != nil {
					return handleErr(err, st, wt, statePath)
				}
			}
			_ = state.Write(statePath, st)
			_ = wt.Commit(ctx,
				fmt.Sprintf("chore(migrate-loop): completed %s in %d iters", st.Slug, st.BudgetUsed),
				map[string]string{"EXPECTED_COMMIT_PREFIX": "chore(migrate-loop)"})
			return nil
		case state.PhaseEscalated:
			return nil
		}
		_ = state.Write(statePath, st)
	}
}

// resolveBaseRef picks the base ref for the migrate-loop branch. It prefers
// origin/main (after a best-effort fetch) so the migration is implemented
// against the latest remote state, not whatever stale tip happens to be on
// the user's local main. Falls back to local main if no origin remote is
// configured (e.g. local-only test repos or pre-push workflows).
func resolveBaseRef(ctx context.Context, sourceRepo string) string {
	// Best-effort fetch. Don't hard-fail: a repo with no origin (test fixtures,
	// offline machines) is still usable; we just branch from local main.
	if err := exec.CommandContext(ctx, "git", "-C", sourceRepo,
		"fetch", "origin", "main").Run(); err != nil {
		// Fall through; rev-parse below decides what's actually available.
	}
	// Prefer origin/main if it resolves.
	if err := exec.CommandContext(ctx, "git", "-C", sourceRepo,
		"rev-parse", "--verify", "origin/main").Run(); err == nil {
		return "origin/main"
	}
	return "main"
}

// ensureGitIdentity sets a fallback user.email/user.name in the worktree's
// local config when none is present. This avoids "Please tell me who you are"
// failures during the initial chore commit on fresh test repos.
func ensureGitIdentity(ctx context.Context, dir string) {
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email").Run(); err != nil {
		_ = exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email", "migrate-loop@example.invalid").Run()
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name").Run(); err != nil {
		_ = exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name", "migrate-loop").Run()
	}
}

type EscalationError struct {
	Kind       escalate.Kind
	Underlying error
}

func (e *EscalationError) Error() string {
	return fmt.Sprintf("escalation %s: %v", e.Kind, e.Underlying)
}

func handleErr(err error, st *state.State, wt *worktree.Worktree, statePath string) error {
	kind := classifyEscalation(err)
	if kind == "" {
		return err // Class 1: infra error → exit 1
	}
	st.Phase = state.PhaseEscalated
	st.EscalationReason = string(kind)
	_ = state.Write(statePath, st)
	_ = escalate.Write(filepath.Join(wt.Path, "ESCALATION.md"), escalate.Reason{
		Kind:        kind,
		Summary:     err.Error(),
		LastFailing: st.LastFailing,
	})
	_ = wt.Commit(context.Background(),
		fmt.Sprintf("chore(migrate-loop): escalate %s", kind),
		map[string]string{"EXPECTED_COMMIT_PREFIX": "chore(migrate-loop)"})
	return &EscalationError{Kind: kind, Underlying: err}
}

func classifyEscalation(err error) escalate.Kind {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "tests_vacuous"):
		return escalate.KindTestsVacuous
	case strings.Contains(msg, "budget_exhausted"):
		return escalate.KindBudgetExhausted
	case strings.Contains(msg, "stagnation_after_redirect"):
		return escalate.KindStagnationAfterRedirect
	case strings.Contains(msg, "tests_seem_wrong"):
		return escalate.KindTestsSeemWrong
	case strings.Contains(msg, "timed out"):
		return escalate.KindIterationTimeout
	}
	return ""
}

// IsEscalation reports whether err is an EscalationError (used by main for exit codes and tests).
func IsEscalation(err error) (escalate.Kind, bool) {
	var ee *EscalationError
	if errors.As(err, &ee) {
		return ee.Kind, true
	}
	return "", false
}
