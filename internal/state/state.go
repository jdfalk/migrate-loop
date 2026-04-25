// Package state owns the on-disk loop state (STATE.md).
package state

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Phase int

const (
	PhaseInit Phase = iota
	PhasePlan
	PhaseCode
	PhaseRedirect
	PhaseCover
	PhasePR
	PhaseEscalated
)

func (p Phase) String() string {
	switch p {
	case PhaseInit:
		return "INIT"
	case PhasePlan:
		return "PLAN"
	case PhaseCode:
		return "CODE"
	case PhaseRedirect:
		return "REDIRECT"
	case PhaseCover:
		return "COVER"
	case PhasePR:
		return "PR"
	case PhaseEscalated:
		return "ESCALATED"
	}
	return ""
}

func ParsePhase(s string) (Phase, error) {
	switch strings.ToUpper(s) {
	case "INIT":
		return PhaseInit, nil
	case "PLAN":
		return PhasePlan, nil
	case "CODE":
		return PhaseCode, nil
	case "REDIRECT":
		return PhaseRedirect, nil
	case "COVER":
		return PhaseCover, nil
	case "PR":
		return PhasePR, nil
	case "ESCALATED":
		return PhaseEscalated, nil
	}
	return 0, fmt.Errorf("state: unknown phase %q", s)
}

type TestID struct {
	Package string `yaml:"package"`
	Test    string `yaml:"test"`
}

type OscillationEvent struct {
	Iteration int    `yaml:"iteration"`
	Note      string `yaml:"note"`
}

type State struct {
	SchemaVersion          int                `yaml:"schema_version"`
	Slug                   string             `yaml:"slug"`
	Phase                  Phase              `yaml:"-"`
	PhaseStr               string             `yaml:"phase"`
	Iteration              int                `yaml:"iteration"`
	Budget                 int                `yaml:"budget"`
	CoverageBudget         int                `yaml:"coverage_budget"`
	StagnationStreak       int                `yaml:"stagnation_streak"`
	OscillationLog         []OscillationEvent `yaml:"oscillation_log,omitempty"`
	LastFailing            []TestID           `yaml:"last_failing,omitempty"`
	LastDiffSummary        string             `yaml:"last_diff_summary,omitempty"`
	BudgetUsed             int                `yaml:"budget_used"`
	CoverageBudgetUsed     int                `yaml:"coverage_budget_used"`
	HumanInterventionCount int                `yaml:"human_intervention_count"`
	EscalationReason       string             `yaml:"escalation_reason,omitempty"`
	TotalCostUSD           float64            `yaml:"total_cost_usd"`
	RedirectUsed           bool               `yaml:"redirect_used"`
}

const stateBody = `# Migration State

This file is managed by migrate-loop. Editing manually is allowed but you should
generally re-invoke 'migrate-loop --resume' rather than hand-modify state.
`

func Write(path string, s *State) error {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = 1
	}
	s.PhaseStr = s.Phase.String()
	yml, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}
	out := "---\n" + string(yml) + "---\n" + stateBody
	return os.WriteFile(path, []byte(out), 0o644)
}

func Read(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("state: read %s: %w", path, err)
	}
	src := string(raw)
	if !strings.HasPrefix(src, "---\n") {
		return nil, errors.New("state: STATE.md missing frontmatter")
	}
	rest := strings.TrimPrefix(src, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, errors.New("state: STATE.md frontmatter not terminated")
	}
	var s State
	if err := yaml.Unmarshal([]byte(rest[:end]), &s); err != nil {
		return nil, fmt.Errorf("state: unmarshal: %w", err)
	}
	p, err := ParsePhase(s.PhaseStr)
	if err != nil {
		return nil, err
	}
	s.Phase = p
	return &s, nil
}

type ProgressResult struct {
	IsProgress   bool
	Oscillation  bool // count equal but set differs
	AllGreen     bool
	NowFailing   int
	NowPassing   []TestID // tests that previously failed and now pass
	NewlyFailing []TestID // tests that did not previously fail but do now
}

func DetectProgress(prev, curr []TestID) ProgressResult {
	prevSet := setOf(prev)
	currSet := setOf(curr)
	res := ProgressResult{NowFailing: len(curr)}
	if len(curr) == 0 {
		res.IsProgress = true
		res.AllGreen = true
		return res
	}
	if len(curr) < len(prev) {
		res.IsProgress = true
	}
	for k, id := range prevSet {
		if _, ok := currSet[k]; !ok {
			res.NowPassing = append(res.NowPassing, id)
		}
	}
	for k, id := range currSet {
		if _, ok := prevSet[k]; !ok {
			res.NewlyFailing = append(res.NewlyFailing, id)
		}
	}
	if len(res.NowPassing) > 0 || len(res.NewlyFailing) > 0 {
		res.IsProgress = true
		if len(curr) == len(prev) {
			res.Oscillation = true
		}
	}
	return res
}

func setOf(ids []TestID) map[string]TestID {
	m := make(map[string]TestID, len(ids))
	for _, id := range ids {
		m[id.Package+"::"+id.Test] = id
	}
	return m
}
