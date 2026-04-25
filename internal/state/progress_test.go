package state

import "testing"

func TestProgress_CountDecreased(t *testing.T) {
	prev := []TestID{{Package: "p", Test: "A"}, {Package: "p", Test: "B"}, {Package: "p", Test: "C"}}
	curr := []TestID{{Package: "p", Test: "A"}, {Package: "p", Test: "B"}}
	r := DetectProgress(prev, curr)
	if !r.IsProgress {
		t.Error("expected progress when count decreased")
	}
	if r.Oscillation {
		t.Error("did not expect oscillation")
	}
}

func TestProgress_SetRotated(t *testing.T) {
	prev := []TestID{{Package: "p", Test: "A"}, {Package: "p", Test: "B"}}
	curr := []TestID{{Package: "p", Test: "B"}, {Package: "p", Test: "C"}}
	r := DetectProgress(prev, curr)
	if !r.IsProgress {
		t.Error("expected progress when set rotated")
	}
	if !r.Oscillation {
		t.Error("expected oscillation flag when set rotated but count equal")
	}
}

func TestProgress_NoChange(t *testing.T) {
	prev := []TestID{{Package: "p", Test: "A"}, {Package: "p", Test: "B"}}
	curr := []TestID{{Package: "p", Test: "A"}, {Package: "p", Test: "B"}}
	r := DetectProgress(prev, curr)
	if r.IsProgress {
		t.Error("expected NO progress when nothing changed")
	}
}

func TestProgress_AllGreen(t *testing.T) {
	prev := []TestID{{Package: "p", Test: "A"}}
	curr := []TestID{}
	r := DetectProgress(prev, curr)
	if !r.IsProgress {
		t.Error("expected progress when all green")
	}
	if !r.AllGreen {
		t.Error("expected AllGreen flag")
	}
}
