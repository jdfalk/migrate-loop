// Package escalate writes the on-disk ESCALATION.md document.
package escalate

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/jdfalk/migrate-loop/internal/state"
)

type Kind string

const (
	KindTestsVacuous            Kind = "tests_vacuous"
	KindBudgetExhausted         Kind = "budget_exhausted"
	KindStagnationAfterRedirect Kind = "stagnation_after_redirect"
	KindTestsSeemWrong          Kind = "tests_seem_wrong"
	KindIterationTimeout        Kind = "iteration_timeout"
)

type Reason struct {
	Kind           Kind
	Summary        string
	LastFailing    []state.TestID
	LastTestOutput string
	LastDiffs      []string
	AgentDiagnosis string
	SuggestedFix   string
}

const tmplSrc = `# Migration escalation: {{.Kind}}

**Summary:** {{.Summary}}

## Last failing tests
{{range .LastFailing}}- {{.Package}} :: {{.Test}}
{{end}}
## Last test output

` + "```" + `
{{.LastTestOutput}}
` + "```" + `

## Last 3 diffs
{{range $i, $d := .LastDiffs}}### Diff {{$i}}
` + "```diff" + `
{{$d}}
` + "```" + `
{{end}}
## Agent diagnosis
{{.AgentDiagnosis}}

## Suggested fix
{{if .SuggestedFix}}{{.SuggestedFix}}{{else}}(none){{end}}

---
*To resume after fixing: ` + "`migrate-loop --resume`" + `*
`

func Write(path string, r Reason) error {
	t, err := template.New("esc").Parse(tmplSrc)
	if err != nil {
		return err
	}
	if r.SuggestedFix == "" {
		r.SuggestedFix = defaultSuggestion(r.Kind)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, r); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func defaultSuggestion(k Kind) string {
	switch k {
	case KindTestsVacuous:
		return "Tests do not actually exercise the new behavior; revise spec or planner output."
	case KindBudgetExhausted:
		return "Increase --budget OR re-scope spec into smaller migrations."
	case KindStagnationAfterRedirect:
		return "Tests may be testing the wrong thing, or behavior is under-specified."
	case KindTestsSeemWrong:
		return "Review FROZEN_TESTS.md, decide whether the agent's objection is correct, edit tests if so, then --resume."
	case KindIterationTimeout:
		return "Lower scope or raise --iter-timeout."
	}
	return fmt.Sprintf("(no default suggestion for %s)", k)
}
