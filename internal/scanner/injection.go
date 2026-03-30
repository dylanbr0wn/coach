package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylanbr0wn/coach/pkg"
)

func CheckInjection(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
	var findings []pkg.Finding
	for _, p := range patterns {
		if p.Category != "prompt-injection" || p.Regex == "" {
			continue
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			continue
		}
		if matchesFileType(p.FileTypes, "*.md") {
			matches := re.FindAllStringIndex(s.Body, -1)
			for _, m := range matches {
				line := lineNumber(s.Body, m[0])
				findings = append(findings, pkg.Finding{
					ID:          p.ID,
					Category:    p.Category,
					Severity:    pkg.SeverityFromString(p.Severity),
					Name:        p.Name,
					Description: p.Description,
					File:        filepath.Join(s.Path, "SKILL.md"),
					Line:        line,
					Match:       s.Body[m[0]:m[1]],
				})
			}
		}
	}
	return findings
}

func ScanSkillFiles(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
	var findings []pkg.Finding
	_ = filepath.Walk(s.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, p := range patterns {
			if p.Category != "prompt-injection" || p.Regex == "" || !matchesFileType(p.FileTypes, path) {
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
					Severity:    pkg.SeverityFromString(p.Severity),
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

func matchesFileType(fileTypes []string, filename string) bool {
	if len(fileTypes) == 0 {
		return true
	}
	for _, ft := range fileTypes {
		if matched, _ := filepath.Match(ft, filepath.Base(filename)); matched {
			return true
		}
		if strings.HasPrefix(ft, "*.") {
			ext := ft[1:]
			if strings.HasSuffix(filename, ext) {
				return true
			}
		}
	}
	return false
}

func lineNumber(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}

func FormatMatch(match string) string {
	match = strings.TrimSpace(match)
	if len(match) > 60 {
		return fmt.Sprintf("%s...", match[:57])
	}
	return match
}
