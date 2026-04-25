// Package spec parses migration specs (YAML frontmatter + markdown body).
package spec

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrNoFrontmatter = errors.New("spec: no YAML frontmatter found (expected '---' delimiters at start of file)")

type Spec struct {
	Title           string   `yaml:"title"`
	Slug            string   `yaml:"slug"`
	TargetPackages  []string `yaml:"target_packages"`
	TestRunner      string   `yaml:"test_runner"`
	PriorExamples   []string `yaml:"prior_examples"`
	SuccessCriteria []string `yaml:"success_criteria"`

	Body     string `yaml:"-"` // markdown body after frontmatter
	FilePath string `yaml:"-"` // for resolving relative prior_examples
}

func ParseFile(path string) (*Spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("spec: read %s: %w", path, err)
	}
	s, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("spec %s: %w", path, err)
	}
	s.FilePath = path
	return s, nil
}

func Parse(raw []byte) (*Spec, error) {
	src := string(raw)
	if !strings.HasPrefix(src, "---\n") && !strings.HasPrefix(src, "---\r\n") {
		return nil, ErrNoFrontmatter
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(src, "---\n"), "---\r\n")
	end := findClosingDelimiter(rest)
	if end < 0 {
		return nil, ErrNoFrontmatter
	}
	yamlPart := rest[:end]
	body := strings.TrimPrefix(strings.TrimPrefix(rest[end+4:], "\n"), "\r\n")

	var s Spec
	if err := yaml.Unmarshal([]byte(yamlPart), &s); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	s.Body = body
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Spec) Validate() error {
	if s.Slug == "" {
		return errors.New("spec: 'slug' is required")
	}
	if len(s.TargetPackages) == 0 {
		return errors.New("spec: 'target_packages' must list at least one package")
	}
	if s.TestRunner == "" {
		return errors.New("spec: 'test_runner' is required")
	}
	return nil
}

// findClosingDelimiter returns the index of the "\n---" sequence that ends
// the YAML frontmatter — i.e. one where "---" is on its own line. Returns -1
// if no valid closing delimiter is found.
func findClosingDelimiter(s string) int {
	needle := "\n---"
	i := 0
	for {
		idx := strings.Index(s[i:], needle)
		if idx < 0 {
			return -1
		}
		absolute := i + idx
		after := absolute + len(needle)
		// Valid if followed by EOF, "\n", or "\r\n"
		if after == len(s) ||
			(after < len(s) && s[after] == '\n') ||
			(after+1 < len(s) && s[after] == '\r' && s[after+1] == '\n') {
			return absolute
		}
		i = absolute + 1
	}
}
