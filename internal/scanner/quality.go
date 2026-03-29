package scanner

import (
	"path/filepath"

	"github.com/dylan/coach/pkg"
)

func CheckQuality(s *pkg.Skill) []pkg.Finding {
	var findings []pkg.Finding
	if len(s.AllowedTools) == 0 {
		findings = append(findings, pkg.Finding{
			ID:          "QW-001",
			Category:    "quality",
			Severity:    pkg.SeverityWarning,
			Name:        "Missing allowed-tools",
			Description: "Skill does not declare tool restrictions — consider adding allowed-tools to limit permissions",
			File:        filepath.Join(s.Path, "SKILL.md"),
		})
	}
	if len(s.Description) < 20 {
		findings = append(findings, pkg.Finding{
			ID:          "QW-002",
			Category:    "quality",
			Severity:    pkg.SeverityWarning,
			Name:        "Overly broad description",
			Description: "Skill description is too short to be specific (under 20 characters)",
			File:        filepath.Join(s.Path, "SKILL.md"),
		})
	}
	return findings
}
