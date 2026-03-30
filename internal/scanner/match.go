package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dylanbr0wn/coach/pkg"
)

// source describes a single piece of content to scan.
type source struct {
	content  string
	filePath string
}

// matchPatterns runs all patterns with the given category against each source.
// Regexes are compiled once. Compile errors are collected and returned
// separately so callers can surface them (tests) or discard them (production).
func matchPatterns(
	category string,
	sources []source,
	patterns []pkg.Pattern,
) (findings []pkg.Finding, compileErrs []error) {
	// Filter and compile patterns once.
	type compiled struct {
		pattern pkg.Pattern
		re      *regexp.Regexp
	}
	var ready []compiled
	for _, p := range patterns {
		if p.Category != category || p.Regex == "" {
			continue
		}
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			compileErrs = append(compileErrs, err)
			continue
		}
		ready = append(ready, compiled{pattern: p, re: re})
	}

	for _, src := range sources {
		for _, c := range ready {
			if !matchesFileType(c.pattern.FileTypes, src.filePath) {
				continue
			}
			matches := c.re.FindAllStringIndex(src.content, -1)
			for _, m := range matches {
				findings = append(findings, pkg.Finding{
					ID:          c.pattern.ID,
					Category:    c.pattern.Category,
					Severity:    pkg.SeverityFromString(c.pattern.Severity),
					Name:        c.pattern.Name,
					Description: c.pattern.Description,
					File:        src.filePath,
					Line:        lineNumber(src.content, m[0]),
					Match:       src.content[m[0]:m[1]],
				})
			}
		}
	}
	return findings, compileErrs
}

// walkSources reads all files under dir into source structs.
func walkSources(dir string) ([]source, error) {
	var sources []source
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		sources = append(sources, source{content: string(content), filePath: path})
		return nil
	})
	return sources, err
}

// matchesFileType checks if a filename matches any of the given file type patterns.
// Returns true if fileTypes is empty (matches all).
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

// lineNumber returns the 1-based line number for the given byte offset.
func lineNumber(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}
