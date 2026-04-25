package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/tools/cover"
)

// CoverProfile runs `go test -coverprofile=...` in cwd and parses the resulting
// profile into a CoverageReport. Failing tests are tolerated; only environment
// errors (missing binary, ctx cancel) surface.
func (g *GoRunner) CoverProfile(ctx context.Context, cwd string) (CoverageReport, error) {
	out := filepath.Join(cwd, ".migrate-loop-cover.out")
	defer os.Remove(out)
	c := exec.CommandContext(ctx, "go", "test", "-coverprofile="+out, "./...")
	c.Dir = cwd
	if err := c.Run(); err != nil {
		if ctx.Err() != nil {
			return CoverageReport{}, ctx.Err()
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return CoverageReport{}, fmt.Errorf("coverprofile: %w", err)
		}
	}
	profiles, err := cover.ParseProfiles(out)
	if err != nil {
		return CoverageReport{}, fmt.Errorf("parse cover profile: %w", err)
	}
	rep := CoverageReport{ByFile: map[string]FileCoverage{}}
	for _, p := range profiles {
		fc := FileCoverage{Path: p.FileName}
		for _, b := range p.Blocks {
			if b.Count == 0 {
				for ln := b.StartLine; ln <= b.EndLine; ln++ {
					fc.UncoveredLines = append(fc.UncoveredLines, ln)
				}
			}
		}
		rep.ByFile[p.FileName] = fc
	}
	return rep, nil
}
