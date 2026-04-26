// migrate-loop drives a TDD migration via claude -p.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/jdfalk/migrate-loop/internal/agent"
)

// Build-time version metadata. Populated by GoReleaser via -ldflags
// "-X main.version=... -X main.commit=... -X main.date=...". Defaults
// below apply for `go build` / `go install` from source.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var cfg Config
	var showVersion bool
	flag.StringVar(&cfg.SpecPath, "spec", "", "path to migration spec markdown")
	flag.StringVar(&cfg.SourceRepo, "repo", ".", "path to source git repo")
	flag.StringVar(&cfg.WorktreeDir, "worktree-dir", "", "path for worktree (default: <repo>-migrate-<slug>)")
	flag.IntVar(&cfg.Budget, "budget", 50, "max CODE iterations")
	flag.IntVar(&cfg.CoverageBudget, "coverage-budget", 0, "max COVER iterations (default: ceil(0.3*budget))")
	flag.DurationVar(&cfg.IterTimeout, "iter-timeout", 10*time.Minute, "per-iteration claude -p timeout")
	flag.BoolVar(&cfg.Resume, "resume", false, "resume existing worktree from STATE.md")
	flag.BoolVar(&showVersion, "version", false, "print version information and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("migrate-loop %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		fmt.Printf("  go:     %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}

	if cfg.SpecPath == "" {
		fmt.Fprintln(os.Stderr, "--spec is required (or use --version)")
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
