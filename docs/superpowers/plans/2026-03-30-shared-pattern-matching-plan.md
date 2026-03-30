# Shared Pattern Matching Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract duplicated pattern matching logic from `injection.go` and `script.go` into a shared `match.go`, eliminating code duplication and cross-package coupling.

**Architecture:** A single unexported `matchPatterns` function in `scanner/match.go` handles regex compilation, category filtering, file-type matching, line number calculation, and finding construction. Existing public functions become thin wrappers. `SeverityFromString` moves from `rules` to `pkg` to eliminate the `scanner -> rules` import.

**Tech Stack:** Go 1.24, standard library only (regexp, path/filepath, os, strings)

---

### Task 1: Move `SeverityFromString` to `pkg/types.go`

**Files:**
- Modify: `pkg/types.go:47` (after `ScorePoints` method)
- Modify: `internal/rules/loader.go:12-28` (remove function, update callers)
- Modify: `internal/scanner/injection.go:10` (remove `rules` import)
- Modify: `internal/scanner/script.go:8` (remove `rules` import)

- [ ] **Step 1: Write a test for `SeverityFromString` in `pkg/`**

Create `pkg/types_test.go`:

```go
package pkg

import "testing"

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Severity
	}{
		{"critical", SeverityCritical},
		{"high", SeverityHigh},
		{"medium", SeverityMedium},
		{"warning", SeverityWarning},
		{"info", SeverityInfo},
		{"", SeverityInfo},
		{"unknown", SeverityInfo},
		{"CRITICAL", SeverityInfo}, // case-sensitive, unknown falls to info
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SeverityFromString(tt.input)
			if got != tt.want {
				t.Errorf("SeverityFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./pkg/ -run TestSeverityFromString -v`
Expected: FAIL — `SeverityFromString` not defined in `pkg`

- [ ] **Step 3: Add `SeverityFromString` to `pkg/types.go`**

Add after the `ScorePoints` method (after line 47):

```go
// SeverityFromString converts a string severity to the Severity type.
func SeverityFromString(s string) Severity {
	switch s {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "warning":
		return SeverityWarning
	case "info":
		return SeverityInfo
	default:
		return SeverityInfo
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./pkg/ -run TestSeverityFromString -v`
Expected: PASS

- [ ] **Step 5: Update `rules/loader.go` to call `pkg.SeverityFromString`**

Remove the `SeverityFromString` function definition (lines 12-28). The function is not called within `rules/loader.go` itself — it was only consumed by the `scanner` package. Simply delete lines 12-28 (the function and its comment).

- [ ] **Step 6: Run all tests to verify nothing broke**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./...`
Expected: All tests pass. The `scanner` package still imports `rules` at this point (will be cleaned up in Task 3).

- [ ] **Step 7: Commit**

```bash
git add pkg/types.go pkg/types_test.go internal/rules/loader.go
git commit -m "refactor: move SeverityFromString from rules to pkg (#6)"
```

---

### Task 2: Create `scanner/match.go` with shared matching engine

**Files:**
- Create: `internal/scanner/match.go`
- Create: `internal/scanner/match_test.go`

- [ ] **Step 1: Write tests for `matchPatterns`**

Create `internal/scanner/match_test.go`:

```go
package scanner

import (
	"testing"

	"github.com/dylanbr0wn/coach/pkg"
)

func TestMatchPatternsFindsMatches(t *testing.T) {
	patterns := []pkg.Pattern{
		{
			ID:        "TEST-001",
			Category:  "test-cat",
			Severity:  "high",
			Name:      "Test pattern",
			Description: "Matches foo",
			Regex:     `foo`,
			FileTypes: []string{"*.md"},
		},
	}
	sources := []source{
		{content: "line one\nfoo bar\nline three", filePath: "test.md"},
	}

	findings, compileErrs := matchPatterns("test-cat", sources, patterns)

	if len(compileErrs) != 0 {
		t.Errorf("unexpected compile errors: %v", compileErrs)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.ID != "TEST-001" {
		t.Errorf("ID = %q, want TEST-001", f.ID)
	}
	if f.Line != 2 {
		t.Errorf("Line = %d, want 2", f.Line)
	}
	if f.Match != "foo" {
		t.Errorf("Match = %q, want \"foo\"", f.Match)
	}
	if f.Severity != pkg.SeverityHigh {
		t.Errorf("Severity = %v, want High", f.Severity)
	}
	if f.File != "test.md" {
		t.Errorf("File = %q, want \"test.md\"", f.File)
	}
}

func TestMatchPatternsFiltersByCategory(t *testing.T) {
	patterns := []pkg.Pattern{
		{ID: "A", Category: "cat-a", Severity: "high", Regex: `foo`, FileTypes: []string{"*.md"}},
		{ID: "B", Category: "cat-b", Severity: "high", Regex: `foo`, FileTypes: []string{"*.md"}},
	}
	sources := []source{
		{content: "foo", filePath: "test.md"},
	}

	findings, _ := matchPatterns("cat-a", sources, patterns)

	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1 (only cat-a)", len(findings))
	}
	if findings[0].ID != "A" {
		t.Errorf("ID = %q, want A", findings[0].ID)
	}
}

func TestMatchPatternsCollectsCompileErrors(t *testing.T) {
	patterns := []pkg.Pattern{
		{ID: "BAD", Category: "test", Severity: "high", Regex: `[`, FileTypes: []string{"*.md"}},
		{ID: "GOOD", Category: "test", Severity: "high", Regex: `foo`, FileTypes: []string{"*.md"}},
	}
	sources := []source{
		{content: "foo bar", filePath: "test.md"},
	}

	findings, compileErrs := matchPatterns("test", sources, patterns)

	if len(compileErrs) != 1 {
		t.Errorf("got %d compile errors, want 1", len(compileErrs))
	}
	if len(findings) != 1 {
		t.Errorf("got %d findings, want 1 (from GOOD pattern)", len(findings))
	}
}

func TestMatchPatternsFiltersFileType(t *testing.T) {
	patterns := []pkg.Pattern{
		{ID: "SH-ONLY", Category: "test", Severity: "high", Regex: `foo`, FileTypes: []string{"*.sh"}},
	}
	sources := []source{
		{content: "foo", filePath: "test.md"},
	}

	findings, _ := matchPatterns("test", sources, patterns)

	if len(findings) != 0 {
		t.Errorf("got %d findings, want 0 (file type mismatch)", len(findings))
	}
}

func TestMatchPatternsSkipsEmptyRegex(t *testing.T) {
	patterns := []pkg.Pattern{
		{ID: "EMPTY", Category: "test", Severity: "high", Regex: "", FileTypes: []string{"*.md"}},
	}
	sources := []source{
		{content: "anything", filePath: "test.md"},
	}

	findings, compileErrs := matchPatterns("test", sources, patterns)

	if len(findings) != 0 {
		t.Errorf("got %d findings, want 0", len(findings))
	}
	if len(compileErrs) != 0 {
		t.Errorf("got %d compile errors, want 0", len(compileErrs))
	}
}

func TestMatchPatternsMultipleSources(t *testing.T) {
	patterns := []pkg.Pattern{
		{ID: "P1", Category: "test", Severity: "warning", Regex: `TODO`, FileTypes: []string{"*.go"}},
	}
	sources := []source{
		{content: "TODO: fix this", filePath: "a.go"},
		{content: "no match here", filePath: "b.go"},
		{content: "another TODO", filePath: "c.go"},
	}

	findings, _ := matchPatterns("test", sources, patterns)

	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2", len(findings))
	}
	if findings[0].File != "a.go" {
		t.Errorf("findings[0].File = %q, want a.go", findings[0].File)
	}
	if findings[1].File != "c.go" {
		t.Errorf("findings[1].File = %q, want c.go", findings[1].File)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./internal/scanner/ -run TestMatchPatterns -v`
Expected: FAIL — `source` and `matchPatterns` not defined

- [ ] **Step 3: Implement `match.go`**

Create `internal/scanner/match.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./internal/scanner/ -run TestMatchPatterns -v`
Expected: All 6 `TestMatchPatterns*` tests PASS

- [ ] **Step 5: Run all tests to verify nothing broke**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./...`
Expected: All tests pass. At this point `matchesFileType` and `lineNumber` are defined in both `match.go` and `injection.go` — the build will fail due to redeclaration. Proceed to Task 3 immediately to fix this.

- [ ] **Step 6: Commit (if build passes) or continue to Task 3**

If the build fails due to duplicate symbols, skip the commit — Task 3 will resolve this and commit together.

---

### Task 3: Rewrite wrappers to use `matchPatterns`

**Files:**
- Modify: `internal/scanner/injection.go` (rewrite to thin wrapper)
- Modify: `internal/scanner/script.go` (rewrite to thin wrapper)

- [ ] **Step 1: Rewrite `injection.go`**

Replace the entire contents of `internal/scanner/injection.go` with:

```go
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
```

- [ ] **Step 2: Rewrite `script.go`**

Replace the entire contents of `internal/scanner/script.go` with:

```go
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
```

- [ ] **Step 3: Delete `FormatMatch`**

`FormatMatch` in the old `injection.go` is unused anywhere in the codebase. It was removed when `injection.go` was rewritten in Step 1. No `format.go` file is needed — the function is dead code.

- [ ] **Step 4: Verify the build compiles**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go build ./...`
Expected: Clean build, no errors. The `rules` import is now gone from both `injection.go` and `script.go`.

- [ ] **Step 5: Run all tests**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./...`
Expected: All tests pass — public API unchanged, existing `scanner_test.go` works as before.

- [ ] **Step 6: Commit**

```bash
git add internal/scanner/match.go internal/scanner/match_test.go internal/scanner/injection.go internal/scanner/script.go
git commit -m "refactor: extract shared pattern matching engine (#6)

Consolidate duplicated regex matching loops from injection.go and
script.go into matchPatterns() in match.go. Wrappers become thin
delegates. Regexes compile once per call. Compile errors are collected
instead of silently swallowed."
```

---

### Task 4: Final verification

**Files:** None modified — verification only.

- [ ] **Step 1: Run the full test suite**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go test ./... -v`
Expected: All tests pass across all packages.

- [ ] **Step 2: Run `go vet`**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && go vet ./...`
Expected: No issues.

- [ ] **Step 3: Verify `scanner` no longer imports `rules`**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && grep -r '"github.com/dylanbr0wn/coach/internal/rules"' internal/scanner/`
Expected: No output — the `scanner -> rules` dependency is eliminated.

- [ ] **Step 4: Verify no duplicate function definitions**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/shared-pattern-matching && grep -rn 'func matchesFileType\|func lineNumber\|func FormatMatch' internal/scanner/`
Expected: Each function appears exactly once, in `match.go` only.
