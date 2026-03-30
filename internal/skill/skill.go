package skill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylanbr0wn/coach/pkg"
	"gopkg.in/yaml.v3"
)

// Parse reads a SKILL.md from the given directory and returns a parsed Skill.
func Parse(dir string) (*pkg.Skill, error) {
	skillPath := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("reading SKILL.md: %w", err)
	}

	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	var s pkg.Skill
	if err := yaml.Unmarshal(frontmatter, &s); err != nil {
		return nil, fmt.Errorf("parsing YAML frontmatter: %w", err)
	}

	s.Path = dir
	s.Body = body

	if info, err := os.Stat(filepath.Join(dir, "tests")); err == nil && info.IsDir() {
		s.HasTests = true
	}

	if err := validateRequired(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

// Validate checks a parsed skill for spec compliance issues.
func Validate(s *pkg.Skill) []string {
	var errs []string
	if len(s.Name) > 64 {
		errs = append(errs, fmt.Sprintf("name is %d chars, max is 64", len(s.Name)))
	}
	if !isValidName(s.Name) {
		errs = append(errs, "name must be lowercase alphanumeric with hyphens only")
	}
	if len(s.Description) > 1024 {
		errs = append(errs, fmt.Sprintf("description is %d chars, max is 1024", len(s.Description)))
	}
	if strings.TrimSpace(s.Body) == "" {
		errs = append(errs, "skill body is empty — add instructions below frontmatter")
	}
	return errs
}

func validateRequired(s *pkg.Skill) error {
	if s.Name == "" {
		return fmt.Errorf("required field 'name' is missing from frontmatter")
	}
	if s.Description == "" {
		return fmt.Errorf("required field 'description' is missing from frontmatter")
	}
	return nil
}

func splitFrontmatter(data []byte) ([]byte, string, error) {
	const delimiter = "---"
	content := string(data)
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, delimiter) {
		return nil, "", fmt.Errorf("SKILL.md must start with --- frontmatter delimiter")
	}

	rest := content[len(delimiter):]
	endIdx := strings.Index(rest, "\n"+delimiter)
	if endIdx == -1 {
		return nil, "", fmt.Errorf("missing closing --- frontmatter delimiter")
	}

	fm := rest[:endIdx]
	body := strings.TrimSpace(rest[endIdx+len("\n"+delimiter):])

	return bytes.TrimSpace([]byte(fm)), body, nil
}

func isValidName(name string) bool {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
