//go:build live_api

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdfalk/migrate-loop/internal/agent"
)

// TestLive_TrivialAdd actually invokes `claude -p` against a tiny fixture spec.
// Gated by the `live_api` build tag so it only runs via `make test-live`.
//
// Cost guardrails:
//   - --budget 5, --coverage-budget 2 (caps total claude -p invocations)
//   - 5-minute per-iteration timeout
//
// This test exists primarily as a prompt-drift canary, not a quality gate.
// Failure should prompt: did the prompt templates change in a way that
// breaks the planner's ability to write usable failing tests?
func TestLive_TrivialAdd(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY required for live_api tests")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude binary not on PATH")
	}

	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.md")
	specBody := `---
title: trivial add
slug: trivial-add-live
target_packages: ["."]
test_runner: "go test -json ./..."
prior_examples: []
success_criteria: ["Add returns sum of inputs"]
---
# trivial add

Implement Add(a, b int) int that returns a + b. Cover edge cases:
zero inputs, negative inputs, max int overflow (test should document
expected wrap-around behavior).
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// Bare source repo with main branch.
	srcRepo := filepath.Join(tmp, "src")
	if out, err := exec.Command("git", "init", srcRepo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "live@test"},
		{"config", "user.name", "live"},
		{"commit", "--allow-empty", "-m", "init"},
		{"branch", "-M", "main"},
	} {
		c := exec.Command("git", args...)
		c.Dir = srcRepo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	cfg := Config{
		SpecPath:       specPath,
		SourceRepo:     srcRepo,
		WorktreeDir:    filepath.Join(tmp, "wt"),
		Budget:         5,
		CoverageBudget: 2,
		IterTimeout:    5 * time.Minute,
		Agent:          agent.NewCLIAgent(),
		SkipPR:         true,
	}
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatalf("live run failed: %v", err)
	}
}
