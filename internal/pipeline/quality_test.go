package pipeline

import (
	"testing"

	"github.com/dylanbr0wn/coach/internal/types"
)

func TestCheckQuality_AllPass(t *testing.T) {
	s := &types.Skill{
		Name:         "good-skill",
		Description:  "Use when you need to review Go code for best practices",
		AllowedTools: []string{"Read", "Write"},
		Body:         "# Good Skill\n\n## When to Use\n\nUse this skill when reviewing Go code. It provides detailed feedback on idiomatic patterns, error handling, and performance.",
	}

	result := CheckQuality(s)
	if result.Status != CheckPass {
		t.Errorf("status = %v, want CheckPass; issues: %v", result.Status, result.Issues)
	}
}

func TestCheckQuality_ShortDescription(t *testing.T) {
	s := &types.Skill{
		Name:         "short-desc",
		Description:  "A skill",
		AllowedTools: []string{"Read"},
		Body:         "# Skill\n\n## When to Use\n\nUse this when you need help. This body is long enough to pass the 50-char check easily.",
	}

	result := CheckQuality(s)
	if result.Status != CheckWarn {
		t.Fatalf("status = %v, want CheckWarn", result.Status)
	}
	assertHasIssue(t, result.Issues, "description under 20 chars")
}

func TestCheckQuality_NoTriggerPhrase(t *testing.T) {
	s := &types.Skill{
		Name:         "no-trigger",
		Description:  "Helps with Go code review and best practices",
		AllowedTools: []string{"Read"},
		Body:         "# Skill\n\n## When to Use\n\nReview Go code for idiomatic patterns. This body is long enough to pass the check.",
	}

	result := CheckQuality(s)
	if result.Status != CheckWarn {
		t.Fatalf("status = %v, want CheckWarn", result.Status)
	}
	assertHasIssue(t, result.Issues, "no trigger phrase")
}

func TestCheckQuality_NoAllowedTools(t *testing.T) {
	s := &types.Skill{
		Name:        "no-tools",
		Description: "Use when reviewing Go code for best practices and patterns",
		Body:        "# Skill\n\n## When to Use\n\nReview Go code for idiomatic patterns. This body is long enough to pass the check.",
	}

	result := CheckQuality(s)
	if result.Status != CheckWarn {
		t.Fatalf("status = %v, want CheckWarn", result.Status)
	}
	assertHasIssue(t, result.Issues, "no allowed-tools")
}

func TestCheckQuality_NoWhenToUseSection(t *testing.T) {
	s := &types.Skill{
		Name:         "no-when",
		Description:  "Use when reviewing Go code for best practices and patterns",
		AllowedTools: []string{"Read"},
		Body:         "# Skill\n\nThis skill helps review Go code. It checks for idiomatic patterns and gives feedback on improvements.",
	}

	result := CheckQuality(s)
	if result.Status != CheckWarn {
		t.Fatalf("status = %v, want CheckWarn", result.Status)
	}
	assertHasIssue(t, result.Issues, "no \"When to Use\" section")
}

func TestCheckQuality_ThinBody(t *testing.T) {
	s := &types.Skill{
		Name:         "thin-body",
		Description:  "Use when you need a thin skill for testing",
		AllowedTools: []string{"Read"},
		Body:         "# Thin\n\n## When to Use\n\nShort.",
	}

	result := CheckQuality(s)
	if result.Status != CheckWarn {
		t.Fatalf("status = %v, want CheckWarn", result.Status)
	}
	assertHasIssue(t, result.Issues, "body under 50 chars")
}

func TestCheckQuality_MultipleIssues(t *testing.T) {
	s := &types.Skill{
		Name:        "bad-skill",
		Description: "Bad",
		Body:        "Short.",
	}

	result := CheckQuality(s)
	if result.Status != CheckWarn {
		t.Fatalf("status = %v, want CheckWarn", result.Status)
	}
	if len(result.Issues) != 5 {
		t.Errorf("got %d issues, want 5: %v", len(result.Issues), result.Issues)
	}
}

func assertHasIssue(t *testing.T, issues []string, substr string) {
	t.Helper()
	for _, issue := range issues {
		for i := 0; i <= len(issue)-len(substr); i++ {
			if issue[i:i+len(substr)] == substr {
				return
			}
		}
	}
	t.Errorf("expected issue containing %q, got: %v", substr, issues)
}
