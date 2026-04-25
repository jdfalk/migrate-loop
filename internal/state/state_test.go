package state

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	want := State{
		Slug:             "demo",
		Phase:            PhaseCode,
		Iteration:        7,
		Budget:           50,
		StagnationStreak: 2,
		LastFailing: []TestID{
			{Package: "internal/x", Test: "TestA"},
			{Package: "internal/x", Test: "TestB/sub"},
		},
		OscillationLog: []OscillationEvent{
			{Iteration: 5, Note: "swapped TestA/TestB"},
		},
		BudgetUsed: 7,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "STATE.md")
	if err := Write(path, &want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// SchemaVersion is auto-set by Write to 1; align want for comparison
	want.SchemaVersion = 1
	want.PhaseStr = want.Phase.String()
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("round-trip mismatch:\nwant %+v\ngot  %+v", want, *got)
	}
}

func TestPhaseStringRoundTrip(t *testing.T) {
	cases := []Phase{PhaseInit, PhasePlan, PhaseCode, PhaseRedirect, PhaseCover, PhasePR, PhaseEscalated}
	for _, p := range cases {
		t.Run(p.String(), func(t *testing.T) {
			s := p.String()
			if s == "" {
				t.Errorf("phase %d has empty String()", p)
			}
			got, err := ParsePhase(s)
			if err != nil {
				t.Errorf("ParsePhase(%q): %v", s, err)
			}
			if got != p {
				t.Errorf("ParsePhase round-trip: got %v want %v", got, p)
			}
		})
	}
}

func TestParsePhase_Unknown(t *testing.T) {
	_, err := ParsePhase("BOGUS")
	if err == nil {
		t.Error("expected error for unknown phase")
	}
}
