package scanner

import (
	"path/filepath"

	"github.com/dylanbr0wn/coach/pkg"
)

// CheckInjection scans the skill's markdown body for prompt-injection patterns.
func CheckInjection(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
	src := source{content: s.Body, filePath: filepath.Join(s.Path, "SKILL.md")}
	findings, _ := matchPatterns("prompt-injection", []source{src}, patterns)
	return findings
}

// ScanSkillFiles walks all files in the skill directory and scans for
// prompt-injection patterns. Called separately from ScanSkill by cmd/scan.go
// to provide additional file-level coverage.
func ScanSkillFiles(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
	sources, err := walkSources(s.Path)
	if err != nil {
		return nil
	}
	findings, _ := matchPatterns("prompt-injection", sources, patterns)
	return findings
}
