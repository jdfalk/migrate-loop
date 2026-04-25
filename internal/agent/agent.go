// Package agent abstracts how migrate-loop talks to Claude.
package agent

import (
	"context"
	"time"
)

type Phase string

const (
	PhasePlan     Phase = "PLAN"
	PhaseCode     Phase = "CODE"
	PhaseRedirect Phase = "REDIRECT"
	PhaseCover    Phase = "COVER"
)

type Request struct {
	Phase           Phase
	Cwd             string
	AllowedTools    []string
	DisallowedTools []string
	Env             map[string]string
	Prompt          string
	Timeout         time.Duration
}

type Response struct {
	ExitCode  int
	Stdout    string
	Stderr    string
	Duration  time.Duration
	SessionID string
	Cost      float64
}

type Agent interface {
	Run(ctx context.Context, req Request) (Response, error)
}
