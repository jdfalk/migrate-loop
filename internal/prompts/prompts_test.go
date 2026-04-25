package prompts

import (
	"strings"
	"testing"
)

func TestRenderPlanner(t *testing.T) {
	out, err := RenderPlanner(PlannerInput{
		Slug:           "demo",
		SpecBody:       "# demo\nadd two ints",
		PriorExamples:  []PriorExample{{Path: "p.md", Content: "prior content"}},
		TargetPackages: []string{"internal/x"},
		TestRunner:     "go test -json ./...",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"demo", "add two ints", "prior content", "internal/x", "go test -json"} {
		if !strings.Contains(out, want) {
			t.Errorf("planner prompt missing %q", want)
		}
	}
}

func TestRenderCoder(t *testing.T) {
	out, err := RenderCoder(CoderInput{
		Iteration:       3,
		FailingTests:    "TestA, TestB",
		LastTestOutput:  "--- FAIL: TestA",
		LastDiff:        "+ x := 1",
		OscillationNote: "swapped TestA/TestB last iteration",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"iteration 3", "TestA, TestB", "FROZEN_TESTS.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("coder prompt missing %q\noutput:\n%s", want, out)
		}
	}
}

func TestRenderRedirect(t *testing.T) {
	out, err := RenderRedirect(RedirectInput{
		Iteration:        4,
		OscillationLog:   "iter1: swap; iter2: swap",
		WhatKeepsFailing: "TestX",
		LastDiffs:        "diffs go here",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"STAGNATION", "TestX", "FROZEN_TESTS.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("redirect prompt missing %q", want)
		}
	}
}

func TestRenderCover(t *testing.T) {
	out, err := RenderCover(CoverInput{UncoveredGaps: "- file.go: lines [10 11]"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "file.go") {
		t.Error("cover prompt missing gap content")
	}
}
