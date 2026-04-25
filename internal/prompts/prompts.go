// Package prompts holds embedded prompt templates used by each phase.
package prompts

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed planner.tmpl
var plannerTmpl string

//go:embed coder.tmpl
var coderTmpl string

//go:embed redirect.tmpl
var redirectTmpl string

//go:embed cover.tmpl
var coverTmpl string

// PriorExample is a reference test file shown to the planner for house-style cues.
type PriorExample struct {
	Path    string
	Content string
}

// PlannerInput is the data passed to planner.tmpl.
type PlannerInput struct {
	Slug           string
	SpecBody       string
	PriorExamples  []PriorExample
	TargetPackages []string
	TestRunner     string
}

// RenderPlanner renders the planner prompt.
func RenderPlanner(in PlannerInput) (string, error) {
	return render(plannerTmpl, in)
}

// CoderInput is the data passed to coder.tmpl.
type CoderInput struct {
	Iteration       int
	FailingTests    string
	LastTestOutput  string
	LastDiff        string
	OscillationNote string
}

// RenderCoder renders the per-iteration coder prompt.
func RenderCoder(in CoderInput) (string, error) {
	return render(coderTmpl, in)
}

// RedirectInput is the data passed to redirect.tmpl.
type RedirectInput struct {
	Iteration        int
	OscillationLog   string
	WhatKeepsFailing string
	LastDiffs        string
}

// RenderRedirect renders the stagnation-redirect prompt.
func RenderRedirect(in RedirectInput) (string, error) {
	return render(redirectTmpl, in)
}

// CoverInput is the data passed to cover.tmpl.
type CoverInput struct {
	UncoveredGaps string
}

// RenderCover renders the coverage-planner prompt.
func RenderCover(in CoverInput) (string, error) {
	return render(coverTmpl, in)
}

func render(tmplSrc string, data any) (string, error) {
	t, err := template.New("p").Parse(tmplSrc)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
