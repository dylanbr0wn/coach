package scanner

import (
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/types"
)

func CheckQuality(s *types.Skill) []types.Finding {
	var findings []types.Finding
	if len(s.AllowedTools) == 0 {
		findings = append(findings, types.Finding{
			ID:          "QW-001",
			Category:    "quality",
			Severity:    types.SeverityWarning,
			Name:        "Missing allowed-tools",
			Description: "Skill does not declare tool restrictions — consider adding allowed-tools to limit permissions",
			File:        filepath.Join(s.Path, "SKILL.md"),
		})
	}
	if len(s.Description) < 20 {
		findings = append(findings, types.Finding{
			ID:          "QW-002",
			Category:    "quality",
			Severity:    types.SeverityWarning,
			Name:        "Overly broad description",
			Description: "Skill description is too short to be specific (under 20 characters)",
			File:        filepath.Join(s.Path, "SKILL.md"),
		})
	}
	return findings
}
