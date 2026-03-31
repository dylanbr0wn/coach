package scanner

import (
	"testing"

	"github.com/dylanbr0wn/coach/internal/types"
)

func TestMatchPatternsFindsMatches(t *testing.T) {
	patterns := []types.Pattern{
		{
			ID:          "TEST-001",
			Category:    "test-cat",
			Severity:    "high",
			Name:        "Test pattern",
			Description: "Matches foo",
			Regex:       `foo`,
			FileTypes:   []string{"*.md"},
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

	t.Run("ID", func(t *testing.T) {
		if f.ID != "TEST-001" {
			t.Errorf("matchPatterns().ID = %q, want %q", f.ID, "TEST-001")
		}
	})
	t.Run("Line", func(t *testing.T) {
		if f.Line != 2 {
			t.Errorf("matchPatterns().Line = %d, want %d", f.Line, 2)
		}
	})
	t.Run("Match", func(t *testing.T) {
		if f.Match != "foo" {
			t.Errorf("matchPatterns().Match = %q, want %q", f.Match, "foo")
		}
	})
	t.Run("Severity", func(t *testing.T) {
		if f.Severity != types.SeverityHigh {
			t.Errorf("matchPatterns().Severity = %v, want %v", f.Severity, types.SeverityHigh)
		}
	})
	t.Run("File", func(t *testing.T) {
		if f.File != "test.md" {
			t.Errorf("matchPatterns().File = %q, want %q", f.File, "test.md")
		}
	})
}

func TestMatchPatternsFiltersByCategory(t *testing.T) {
	patterns := []types.Pattern{
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
	patterns := []types.Pattern{
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
	patterns := []types.Pattern{
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
	patterns := []types.Pattern{
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
	patterns := []types.Pattern{
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
