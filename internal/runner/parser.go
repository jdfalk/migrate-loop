package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jdfalk/migrate-loop/internal/state"
)

type goEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// ParseTestJSON parses the newline-delimited JSON output of `go test -json`
// into a Result. Parent tests whose failure is solely propagation from a
// failing subtest are collapsed into the leaf subtest to avoid duplicate
// reporting.
func ParseTestJSON(raw []byte) (Result, error) {
	res := Result{Raw: raw}
	dec := json.NewDecoder(bytes.NewReader(raw))
	type key struct{ pkg, test string }
	finalAction := map[key]string{}
	for dec.More() {
		var e goEvent
		if err := dec.Decode(&e); err != nil {
			return res, fmt.Errorf("runner: decode: %w", err)
		}
		switch e.Action {
		case "fail", "pass", "skip":
			if e.Test != "" {
				finalAction[key{e.Package, e.Test}] = e.Action
			}
		case "build-fail":
			res.Errors = append(res.Errors, fmt.Sprintf("build-fail %s: %s", e.Package, e.Output))
		case "output":
			if isPanic(e.Output) {
				res.Errors = append(res.Errors, fmt.Sprintf("panic in %s/%s: %s", e.Package, e.Test, e.Output))
			}
		}
	}
	// Identify parent tests that have a failing subtest; the parent's "fail"
	// is just propagation, so we drop it and keep the leaf.
	failingSubParents := map[key]bool{}
	for k, action := range finalAction {
		if action != "fail" {
			continue
		}
		if i := strings.Index(k.test, "/"); i > 0 {
			failingSubParents[key{k.pkg, k.test[:i]}] = true
		}
	}
	for k, action := range finalAction {
		id := state.TestID{Package: k.pkg, Test: k.test}
		switch action {
		case "fail":
			if failingSubParents[k] {
				continue
			}
			res.Failing = append(res.Failing, id)
		case "pass":
			res.Passing = append(res.Passing, id)
		}
	}
	return res, nil
}

func isPanic(s string) bool {
	return strings.HasPrefix(s, "panic:")
}
