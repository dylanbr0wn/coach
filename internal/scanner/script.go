package scanner

import (
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/pkg"
)

// CheckScripts scans the skill's scripts/ directory for dangerous patterns.
func CheckScripts(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
	scriptsDir := filepath.Join(s.Path, "scripts")
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return nil
	}
	sources, err := walkSources(scriptsDir)
	if err != nil {
		return nil
	}
	findings, _ := matchPatterns("script-danger", sources, patterns)
	return findings
}
