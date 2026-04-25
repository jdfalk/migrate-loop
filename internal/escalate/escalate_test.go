package escalate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jdfalk/migrate-loop/internal/state"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	r := Reason{
		Kind:           KindBudgetExhausted,
		Summary:        "budget hit at iteration 50",
		LastFailing:    []state.TestID{{Package: "p", Test: "TestX"}},
		LastTestOutput: "--- FAIL: TestX\n  expected 3 got 4",
		LastDiffs:      []string{"diff 1", "diff 2", "diff 3"},
		AgentDiagnosis: "I cannot find a way to make TestX pass without changing TestY",
	}
	path := filepath.Join(dir, "ESCALATION.md")
	if err := Write(path, r); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{"budget_exhausted", "TestX", "expected 3 got 4", "diff 2", "cannot find"} {
		if !strings.Contains(s, want) {
			t.Errorf("ESCALATION.md missing %q\nbody:\n%s", want, s)
		}
	}
}

func TestWrite_AllKinds(t *testing.T) {
	cases := []Kind{
		KindTestsVacuous, KindBudgetExhausted, KindStagnationAfterRedirect,
		KindTestsSeemWrong, KindIterationTimeout,
	}
	for _, k := range cases {
		t.Run(string(k), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ESCALATION.md")
			err := Write(path, Reason{Kind: k, Summary: "test"})
			if err != nil {
				t.Fatalf("Write %s: %v", k, err)
			}
			body, _ := os.ReadFile(path)
			if !strings.Contains(string(body), string(k)) {
				t.Errorf("body missing kind %s", k)
			}
		})
	}
}
