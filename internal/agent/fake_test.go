package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFakeAgent_RunsResponsesInOrder(t *testing.T) {
	calls := 0
	dir := t.TempDir()
	fa := &FakeAgent{
		Responses: []Response{
			{ExitCode: 0, Stdout: "first"},
			{ExitCode: 0, Stdout: "second"},
		},
		Editor: func(req Request) error {
			calls++
			return os.WriteFile(filepath.Join(req.Cwd, "f.txt"), []byte("hi"), 0o644)
		},
	}
	r1, err := fa.Run(context.Background(), Request{Phase: PhasePlan, Cwd: dir})
	if err != nil || r1.Stdout != "first" {
		t.Fatalf("first call: %v %+v", err, r1)
	}
	r2, _ := fa.Run(context.Background(), Request{Phase: PhaseCode, Cwd: dir})
	if r2.Stdout != "second" {
		t.Fatalf("second call stdout = %q", r2.Stdout)
	}
	if calls != 2 {
		t.Errorf("Editor calls = %d, want 2", calls)
	}
	if _, err := os.Stat(filepath.Join(dir, "f.txt")); err != nil {
		t.Errorf("Editor side-effect missing: %v", err)
	}
}

func TestFakeAgent_Exhausted(t *testing.T) {
	fa := &FakeAgent{Responses: []Response{{ExitCode: 0}}}
	_, _ = fa.Run(context.Background(), Request{Cwd: t.TempDir()})
	_, err := fa.Run(context.Background(), Request{Cwd: t.TempDir()})
	if err == nil {
		t.Error("expected error after responses exhausted")
	}
}

func TestFakeAgent_RecordsCalls(t *testing.T) {
	fa := &FakeAgent{Responses: []Response{{ExitCode: 0}, {ExitCode: 0}}}
	fa.Run(context.Background(), Request{Phase: PhasePlan, Prompt: "p1"})
	fa.Run(context.Background(), Request{Phase: PhaseCode, Prompt: "p2"})
	if len(fa.Calls) != 2 {
		t.Fatalf("Calls len = %d", len(fa.Calls))
	}
	if fa.Calls[0].Phase != PhasePlan || fa.Calls[1].Phase != PhaseCode {
		t.Errorf("recorded phases wrong: %+v", fa.Calls)
	}
}
