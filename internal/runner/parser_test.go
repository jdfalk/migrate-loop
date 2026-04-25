package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTestJSON(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := ParseTestJSON(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(res.Failing) != 2 {
		t.Errorf("Failing = %d, want 2: %+v", len(res.Failing), res.Failing)
	}
	if len(res.Passing) < 1 {
		t.Errorf("Passing should be >0; got %d", len(res.Passing))
	}
	hasSubtest := false
	for _, id := range res.Failing {
		if id.Test == "TestB/sub_one" {
			hasSubtest = true
		}
	}
	if !hasSubtest {
		t.Errorf("expected subtest TestB/sub_one in Failing; got %+v", res.Failing)
	}
}

func TestParseTestJSON_BuildFailure(t *testing.T) {
	raw := []byte(`{"Time":"2026-01-01T00:00:00Z","Action":"build-fail","Package":"x","Output":"compile error\n"}` + "\n")
	res, err := ParseTestJSON(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(res.Errors) != 1 {
		t.Errorf("Errors = %d, want 1", len(res.Errors))
	}
}
