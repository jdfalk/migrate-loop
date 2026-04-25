package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_EndToEndHappy(t *testing.T) {
	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.md")
	specBody := `---
title: trivial add
slug: trivial-add
target_packages: ["."]
test_runner: "go test -json ./..."
prior_examples: []
success_criteria: ["all tests pass"]
---
# trivial add
Add two ints.
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatal(err)
	}

	srcRepo := filepath.Join(tmp, "src")
	mustGit(t, tmp, "init", srcRepo)
	mustGit(t, srcRepo, "config", "user.email", "t@t")
	mustGit(t, srcRepo, "config", "user.name", "t")
	mustGit(t, srcRepo, "commit", "--allow-empty", "-m", "init")
	mustGit(t, srcRepo, "branch", "-M", "main")

	cfg := Config{
		SpecPath:       specPath,
		SourceRepo:     srcRepo,
		WorktreeDir:    filepath.Join(tmp, "wt"),
		Budget:         5,
		CoverageBudget: 2,
		IterTimeout:    10 * time.Second,
		Agent:          scriptedHappyAgent(t),
		SkipPR:         true,
	}
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_VacuousTestsEscalates(t *testing.T) {
	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.md")
	specBody := `---
title: trivial
slug: trivial-vacuous
target_packages: ["."]
test_runner: "go test -json ./..."
prior_examples: []
success_criteria: []
---
# trivial
nothing useful
`
	if err := os.WriteFile(specPath, []byte(specBody), 0o644); err != nil {
		t.Fatal(err)
	}

	srcRepo := filepath.Join(tmp, "src")
	mustGit(t, tmp, "init", srcRepo)
	mustGit(t, srcRepo, "config", "user.email", "t@t")
	mustGit(t, srcRepo, "config", "user.name", "t")
	mustGit(t, srcRepo, "commit", "--allow-empty", "-m", "init")
	mustGit(t, srcRepo, "branch", "-M", "main")

	// Planner writes only-passing tests → tests_vacuous escalation.
	fa := vacuousAgent(t)

	cfg := Config{
		SpecPath:       specPath,
		SourceRepo:     srcRepo,
		WorktreeDir:    filepath.Join(tmp, "wt"),
		Budget:         5,
		CoverageBudget: 2,
		IterTimeout:    10 * time.Second,
		Agent:          fa,
		SkipPR:         true,
	}
	err := Run(context.Background(), cfg)
	kind, ok := IsEscalation(err)
	if !ok {
		t.Fatalf("expected EscalationError, got %v", err)
	}
	if string(kind) != "tests_vacuous" {
		t.Errorf("kind = %q, want tests_vacuous", kind)
	}
	// ESCALATION.md should have been written
	if _, statErr := os.Stat(filepath.Join(cfg.WorktreeDir, "ESCALATION.md")); statErr != nil {
		t.Errorf("ESCALATION.md missing: %v", statErr)
	}
}
