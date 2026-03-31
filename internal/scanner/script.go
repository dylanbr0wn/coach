package scanner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/types"
)

// CheckScripts scans the skill's scripts/ directory for dangerous patterns.
func CheckScripts(s *types.Skill, patterns []types.Pattern) ([]types.Finding, error) {
	scriptsDir := filepath.Join(s.Path, "scripts")
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return nil, nil
	}
	sources, err := walkSources(scriptsDir)
	if err != nil {
		return nil, fmt.Errorf("walking scripts in %s: %w", scriptsDir, err)
	}
	findings, _ := matchPatterns("script-danger", sources, patterns)
	return findings, nil
}
