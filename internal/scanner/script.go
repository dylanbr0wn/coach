package scanner

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/dylan/coach/internal/rules"
	"github.com/dylan/coach/pkg"
)

func CheckScripts(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
	var findings []pkg.Finding
	scriptsDir := filepath.Join(s.Path, "scripts")
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return findings
	}
	filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, p := range patterns {
			if p.Category != "script-danger" || p.Regex == "" || !matchesFileType(p.FileTypes, path) {
				continue
			}
			re, compErr := regexp.Compile(p.Regex)
			if compErr != nil {
				continue
			}
			matches := re.FindAllStringIndex(string(content), -1)
			for _, m := range matches {
				line := lineNumber(string(content), m[0])
				findings = append(findings, pkg.Finding{
					ID:          p.ID,
					Category:    p.Category,
					Severity:    rules.SeverityFromString(p.Severity),
					Name:        p.Name,
					Description: p.Description,
					File:        path,
					Line:        line,
					Match:       string(content[m[0]:m[1]]),
				})
			}
		}
		return nil
	})
	return findings
}
