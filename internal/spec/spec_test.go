package spec

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_Valid(t *testing.T) {
	got, err := ParseFile(filepath.Join("testdata", "valid.md"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if got.Slug != "trivial-add" {
		t.Errorf("Slug = %q, want %q", got.Slug, "trivial-add")
	}
	if len(got.TargetPackages) != 2 {
		t.Errorf("TargetPackages len = %d, want 2", len(got.TargetPackages))
	}
	if got.TestRunner != "go test -race -json ./..." {
		t.Errorf("TestRunner = %q", got.TestRunner)
	}
	if !strings.Contains(got.Body, "## Behavior") {
		t.Errorf("Body does not contain expected heading; got: %q", got.Body)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	_, err := ParseFile(filepath.Join("testdata", "no-frontmatter.md"))
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Fatalf("err = %v, want ErrNoFrontmatter", err)
	}
}

func TestParse_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		spec string
		want string // substring expected in error
	}{
		{
			name: "missing slug",
			spec: "---\ntitle: x\ntarget_packages: [\"x\"]\ntest_runner: \"go test\"\n---\nbody",
			want: "slug",
		},
		{
			name: "empty target_packages",
			spec: "---\nslug: x\ntarget_packages: []\ntest_runner: \"go test\"\n---\nbody",
			want: "target_packages",
		},
		{
			name: "missing test_runner",
			spec: "---\nslug: x\ntarget_packages: [\"x\"]\n---\nbody",
			want: "test_runner",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.spec))
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want substring %q", err, tc.want)
			}
		})
	}
}

