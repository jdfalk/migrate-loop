// Package runner abstracts the test runner so the harness can target Go today
// and other ecosystems later.
package runner

import (
	"context"

	"github.com/jdfalk/migrate-loop/internal/state"
)

type Result struct {
	Failing []state.TestID
	Passing []state.TestID
	Errors  []string // build failures, panics
	Raw     []byte
}

type CoverageReport struct {
	ByFile map[string]FileCoverage
}

type FileCoverage struct {
	Path           string
	UncoveredLines []int // line numbers with 0 hits among Go statements
}

type Runner interface {
	Run(ctx context.Context, cwd string) (Result, error)
	CoverProfile(ctx context.Context, cwd string) (CoverageReport, error)
}
