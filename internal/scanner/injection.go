package scanner

import (
	"fmt"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/types"
)

// CheckInjection scans the skill's markdown body for prompt-injection patterns.
func CheckInjection(s *types.Skill, patterns []types.Pattern) []types.Finding {
	src := source{content: s.Body, filePath: filepath.Join(s.Path, "SKILL.md")}
	findings, _ := matchPatterns("prompt-injection", []source{src}, patterns)
	return findings
}

// ScanSkillFiles walks all files in the skill directory and scans for
// prompt-injection patterns. Called separately from ScanSkill by cmd/scan.go
// to provide additional file-level coverage.
func ScanSkillFiles(s *types.Skill, patterns []types.Pattern) ([]types.Finding, error) {
	sources, err := walkSources(s.Path)
	if err != nil {
		return nil, fmt.Errorf("walking skill files in %s: %w", s.Path, err)
	}
	findings, _ := matchPatterns("prompt-injection", sources, patterns)
	return findings, nil
}
