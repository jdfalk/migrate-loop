// Package worktree manages git worktree creation, branch ops, and the
// pre-commit hook that enforces test-freeze.
package worktree

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Options configures a worktree Create call.
type Options struct {
	SourceRepo  string // path to existing repo (must contain BaseRef)
	BranchName  string // e.g. "migrate/aijobs-batch-migration"
	WorktreeDir string // absolute path
	BaseRef     string // e.g. "origin/main"
}

// Worktree represents a created git worktree.
type Worktree struct {
	Path       string
	BranchName string
}

// Create adds a new git worktree at opt.WorktreeDir on a fresh branch
// opt.BranchName starting at opt.BaseRef.
func Create(ctx context.Context, opt Options) (*Worktree, error) {
	if err := os.MkdirAll(filepath.Dir(opt.WorktreeDir), 0o755); err != nil {
		return nil, err
	}
	args := []string{"worktree", "add", opt.WorktreeDir, "-b", opt.BranchName, opt.BaseRef}
	if err := runGit(ctx, opt.SourceRepo, args...); err != nil {
		return nil, fmt.Errorf("worktree add: %w", err)
	}
	return &Worktree{Path: opt.WorktreeDir, BranchName: opt.BranchName}, nil
}

// Commit stages all changes in the worktree and creates a commit with the
// provided message. Optional env overrides are passed to git (e.g. for
// GIT_AUTHOR_* fields).
func (w *Worktree) Commit(ctx context.Context, message string, env map[string]string) error {
	if err := runGitEnv(ctx, w.Path, env, "add", "-A"); err != nil {
		return err
	}
	return runGitEnv(ctx, w.Path, env, "commit", "-m", message)
}

// Push pushes the worktree's branch to origin, setting upstream.
func (w *Worktree) Push(ctx context.Context) error {
	return runGit(ctx, w.Path, "push", "-u", "origin", w.BranchName)
}

// DiffSummary returns `git diff --shortstat refspec` output for the worktree.
func (w *Worktree) DiffSummary(ctx context.Context, refspec string) (string, error) {
	return captureGit(ctx, w.Path, "diff", "--shortstat", refspec)
}

// Remove force-removes the git worktree and deletes the local branch.
// Safe to call even if the worktree or branch doesn't fully exist.
func Remove(ctx context.Context, sourceRepo, worktreeDir, branchName string) error {
	// --force allows removal of worktrees with uncommitted changes.
	_ = runGit(ctx, sourceRepo, "worktree", "remove", "--force", worktreeDir)
	// Delete the local branch; ignore "not found" errors.
	_ = runGit(ctx, sourceRepo, "branch", "-D", branchName)
	return nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	return runGitEnv(ctx, dir, nil, args...)
}

func runGitEnv(ctx context.Context, dir string, env map[string]string, args ...string) error {
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), envSlice(env)...)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v: %w\n%s", args, err, out)
	}
	return nil
}

func captureGit(ctx context.Context, dir string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	var buf bytes.Buffer
	c.Stdout = &buf
	if err := c.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func envSlice(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
