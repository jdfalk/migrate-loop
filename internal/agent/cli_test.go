package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const shimSrc = `#!/usr/bin/env bash
echo '{"session_id":"abc","total_cost_usd":0.42}'
exit 0
`

func TestCLIAgent_ParsesJSON(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte(shimSrc), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &CLIAgent{Binary: bin}
	res, err := a.Run(context.Background(), Request{
		Phase: PhasePlan, Cwd: dir, Prompt: "hi", Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.SessionID != "abc" {
		t.Errorf("SessionID = %q", res.SessionID)
	}
	if res.Cost != 0.42 {
		t.Errorf("Cost = %v", res.Cost)
	}
}

func TestCLIAgent_TimeoutPropagates(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\nsleep 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &CLIAgent{Binary: bin}
	_, err := a.Run(context.Background(), Request{
		Phase: PhasePlan, Cwd: dir, Prompt: "hi", Timeout: 200 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error")
	}
}
