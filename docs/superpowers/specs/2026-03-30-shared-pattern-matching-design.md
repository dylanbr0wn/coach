# Shared Pattern Matching Engine

**Issue:** #6
**Date:** 2026-03-30
**Status:** Design approved

## Problem

`scanner/injection.go` and `scanner/script.go` contain near-identical pattern matching loops (~60 lines each) that differ only in category filter and content source. Additional issues:

- Silent regex compile error swallowing
- Cross-package coupling (`scanner` imports `rules` for `SeverityFromString`)
- Per-file regex recompilation in `script.go` (inconsistent with `injection.go`)

## Approach

Extract a single `matchPatterns` function into `scanner/match.go`. `CheckInjection`, `CheckScripts`, and `ScanSkillFiles` become thin wrappers that build `[]source` and delegate. Move `SeverityFromString` to `pkg/types.go` to eliminate the `scanner -> rules` dependency.

## Design

### New file: `scanner/match.go`

Contains the shared matching engine and helpers:

```go
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
) (findings []pkg.Finding, compileErrs []error)
```

The function:
1. Filters patterns by category once up front
2. Compiles each matching pattern's regex once, collecting compile errors
3. Iterates sources, runs each compiled regex against the content
4. Builds `Finding` structs using `pkg.SeverityFromString`
5. Uses `matchesFileType` and `lineNumber` helpers (also in this file)

A private `walkSources(dir string) ([]source, error)` helper walks a directory with `filepath.Walk`, skips directories and unreadable files, reads each file's content, and returns `[]source` structs.

### Simplified wrappers

**`injection.go`** shrinks to two thin functions:

```go
func CheckInjection(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
    src := source{content: s.Body, filePath: filepath.Join(s.Path, "SKILL.md")}
    findings, _ := matchPatterns("prompt-injection", []source{src}, patterns)
    return findings
}

func ScanSkillFiles(s *pkg.Skill, patterns []pkg.Pattern) []pkg.Finding {
    sources, err := walkSources(s.Path)
    if err != nil {
        return nil
    }
    findings, _ := matchPatterns("prompt-injection", sources, patterns)
    return findings
}
```

**`script.go`** becomes:

```go
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

Both wrappers discard compile errors (matching current behavior). Errors are available for tests to assert on.

### `FormatMatch` removal

`FormatMatch` is dead code — defined in `injection.go` but never called anywhere in the codebase. It is deleted as part of the `injection.go` rewrite.

### `SeverityFromString` move

`SeverityFromString` moves from `rules/loader.go` to `pkg/types.go`. This eliminates the `scanner -> rules` import. `rules/loader.go` updates to call `pkg.SeverityFromString`.

## Files changed

| File | Change |
|------|--------|
| `scanner/match.go` | **New** — `source`, `matchPatterns`, `walkSources`, `matchesFileType`, `lineNumber` |
| `scanner/match_test.go` | **New** — unit tests for `matchPatterns` |
| `scanner/injection.go` | **Simplified** — thin wrapper, helpers removed, dead `FormatMatch` deleted |
| `scanner/script.go` | **Simplified** — thin wrapper |
| `pkg/types.go` | **Added** `SeverityFromString` |
| `rules/loader.go` | **Updated** calls to `pkg.SeverityFromString`, removed function |

## Files unchanged

- `scanner/scanner.go` — still calls `CheckInjection` and `CheckScripts`
- `cmd/scan.go` — still calls `ScanSkillFiles` separately and deduplicates
- `scanner/quality.go`, `scanner/score.go` — untouched

## Testing

- **`scanner/match_test.go`**: Test `matchPatterns` directly with inline `source` structs:
  - Valid patterns produce expected findings
  - Invalid regex (e.g., `"["`) populates `compileErrs` while valid patterns still match
  - Category filtering ignores non-matching patterns
  - `matchesFileType` filtering works correctly
- **Existing `scanner_test.go`**: Passes unchanged — public API is untouched
- **`pkg/types_test.go`**: Test `SeverityFromString` edge cases
