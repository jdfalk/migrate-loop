// migrate-loop drives a TDD migration via claude -p.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jdfalk/migrate-loop/internal/agent"
)

// Build-time version metadata. Populated by GoReleaser via -ldflags
// "-X main.version=... -X main.commit=... -X main.date=...". For
// `go install module@version` builds, versionInfo() falls back to
// runtime/debug.ReadBuildInfo() so the module version still surfaces.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// versionInfo returns the version, commit, and date strings to display.
// Resolution order:
//  1. ldflags-injected vars (set by GoReleaser).
//  2. runtime/debug.BuildInfo (set by `go install module@vX.Y.Z` and by
//     `go build` inside a git working tree on Go 1.18+).
//  3. The "dev"/"unknown" defaults.
func versionInfo() (v, c, d string) {
	v, c, d = version, commit, date
	if v != "dev" && c != "unknown" {
		// ldflags populated everything; trust them.
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		v = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if c == "unknown" && s.Value != "" {
				c = s.Value
				if len(c) > 12 {
					c = c[:12]
				}
			}
		case "vcs.time":
			if d == "unknown" && s.Value != "" {
				d = s.Value
			}
		}
	}
	return
}

func main() {
	var cfg Config
	var showVersion bool
	var overwrite bool
	flag.StringVar(&cfg.SpecPath, "spec", "", "path to migration spec markdown")
	flag.StringVar(&cfg.SourceRepo, "repo", ".", "path to source git repo")
	flag.StringVar(&cfg.WorktreeDir, "worktree-dir", "", "path for worktree (default: <repo>-migrate-<slug>)")
	flag.IntVar(&cfg.Budget, "budget", 50, "max CODE iterations")
	flag.IntVar(&cfg.CoverageBudget, "coverage-budget", 0, "max COVER iterations (default: ceil(0.3*budget))")
	flag.DurationVar(&cfg.IterTimeout, "iter-timeout", 10*time.Minute, "per-iteration claude -p timeout")
	flag.BoolVar(&cfg.Resume, "resume", false, "resume existing worktree from STATE.md")
	flag.BoolVar(&overwrite, "overwrite", false, "destroy existing worktree+branch and start fresh (prompts for confirmation)")
	flag.BoolVar(&showVersion, "version", false, "print version information and exit")
	flag.Parse()

	if showVersion {
		v, c, d := versionInfo()
		fmt.Printf("migrate-loop %s\n", v)
		fmt.Printf("  commit: %s\n", c)
		fmt.Printf("  built:  %s\n", d)
		fmt.Printf("  go:     %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}

	if overwrite {
		cfg.Overwrite = confirmOverwrite(cfg.SpecPath, cfg.WorktreeDir)
		if !cfg.Overwrite {
			fmt.Fprintln(os.Stderr, "aborted")
			os.Exit(1)
		}
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

// confirmOverwrite prints what will be destroyed and asks the user to type
// "yes" to confirm. Returns true only on explicit confirmation.
func confirmOverwrite(specPath, worktreeDir string) bool {
	fmt.Fprintf(os.Stderr, "\n⚠️  --overwrite will permanently destroy:\n")
	fmt.Fprintf(os.Stderr, "   worktree : %s\n", worktreeDir)
	fmt.Fprintf(os.Stderr, "   branch   : migrate/<slug> (derived from %s)\n", specPath)
	fmt.Fprintf(os.Stderr, "   STATE.md, ESCALATION.md, and all uncommitted work in that worktree\n\n")
	fmt.Fprintf(os.Stderr, "Are you really, really sure? Type \"yes\" to confirm: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	return strings.TrimSpace(scanner.Text()) == "yes"
}
