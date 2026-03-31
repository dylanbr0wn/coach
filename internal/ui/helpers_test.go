package ui

import (
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	got := Success("Skill created")
	if !strings.Contains(got, "✓") {
		t.Errorf("Success() missing checkmark icon, got: %q", got)
	}
	if !strings.Contains(got, "Skill created") {
		t.Errorf("Success() missing message, got: %q", got)
	}
}

func TestWarn(t *testing.T) {
	got := Warn("No agents configured")
	if !strings.Contains(got, "⚠") {
		t.Errorf("Warn() missing warning icon, got: %q", got)
	}
	if !strings.Contains(got, "No agents configured") {
		t.Errorf("Warn() missing message, got: %q", got)
	}
}

func TestError(t *testing.T) {
	got := Error("Skill not found", "Run 'coach list' to see available skills")
	if !strings.Contains(got, "✗") {
		t.Errorf("Error() missing X icon, got: %q", got)
	}
	if !strings.Contains(got, "Skill not found") {
		t.Errorf("Error() missing message, got: %q", got)
	}
	if !strings.Contains(got, "→") {
		t.Errorf("Error() missing arrow for suggestion, got: %q", got)
	}
	if !strings.Contains(got, "Run 'coach list'") {
		t.Errorf("Error() missing suggestion, got: %q", got)
	}
}

func TestErrorNoSuggestion(t *testing.T) {
	got := Error("Something broke", "")
	if strings.Contains(got, "→") {
		t.Errorf("Error() with empty suggestion should not contain arrow, got: %q", got)
	}
	if !strings.Contains(got, "✗") {
		t.Errorf("Error() missing X icon, got: %q", got)
	}
}

func TestNextStep(t *testing.T) {
	got := NextStep("generate my-skill", "flesh out the skill with an LLM")
	if !strings.Contains(got, "coach generate my-skill") {
		t.Errorf("NextStep() missing command, got: %q", got)
	}
	if !strings.Contains(got, "flesh out the skill") {
		t.Errorf("NextStep() missing description, got: %q", got)
	}
}
