// migrate-loop drives a TDD migration via claude -p.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jdfalk/migrate-loop/internal/agent"
)

func main() {
	var cfg Config
	flag.StringVar(&cfg.SpecPath, "spec", "", "path to migration spec markdown")
	flag.StringVar(&cfg.SourceRepo, "repo", ".", "path to source git repo")
	flag.StringVar(&cfg.WorktreeDir, "worktree-dir", "", "path for worktree (default: <repo>-migrate-<slug>)")
	flag.IntVar(&cfg.Budget, "budget", 50, "max CODE iterations")
	flag.IntVar(&cfg.CoverageBudget, "coverage-budget", 0, "max COVER iterations (default: ceil(0.3*budget))")
	flag.DurationVar(&cfg.IterTimeout, "iter-timeout", 10*time.Minute, "per-iteration claude -p timeout")
	flag.BoolVar(&cfg.Resume, "resume", false, "resume existing worktree from STATE.md")
	flag.Parse()

	if cfg.SpecPath == "" {
		fmt.Fprintln(os.Stderr, "--spec is required")
		os.Exit(1)
	}
	cfg.Agent = agent.NewCLIAgent()

	err := Run(context.Background(), cfg)
	if err == nil {
		return
	}
	if kind, ok := IsEscalation(err); ok {
		fmt.Fprintf(os.Stderr, "escalation: %s\n", kind)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
