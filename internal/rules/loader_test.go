package rules

import (
	"testing"

	"github.com/dylan/coach/pkg"
)

func TestLoadEmbeddedPatterns(t *testing.T) {
	db, err := LoadPatterns("")
	if err != nil {
		t.Fatalf("LoadPatterns() error: %v", err)
	}
	if len(db.Patterns) == 0 {
		t.Fatal("expected at least one embedded pattern, got 0")
	}

	found := false
	for _, p := range db.Patterns {
		if p.ID == "PI-001" {
			found = true
			if p.Category != "prompt-injection" {
				t.Errorf("PI-001 category = %q, want %q", p.Category, "prompt-injection")
			}
		}
	}
	if !found {
		t.Error("expected pattern PI-001 to exist in embedded patterns")
	}
}

func TestLoadEmbeddedAgents(t *testing.T) {
	reg, err := LoadAgentRegistry("")
	if err != nil {
		t.Fatalf("LoadAgentRegistry() error: %v", err)
	}
	if len(reg.Agents) == 0 {
		t.Fatal("expected at least one embedded agent, got 0")
	}

	cc, ok := reg.Agents["claude-code"]
	if !ok {
		t.Fatal("expected claude-code in agent registry")
	}
	if cc.SkillDir == "" {
		t.Error("claude-code skill_dir should not be empty")
	}
}

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		input string
		want  pkg.Severity
	}{
		{"critical", pkg.SeverityCritical},
		{"high", pkg.SeverityHigh},
		{"medium", pkg.SeverityMedium},
		{"warning", pkg.SeverityWarning},
		{"info", pkg.SeverityInfo},
		{"unknown", pkg.SeverityInfo},
	}
	for _, tt := range tests {
		got := SeverityFromString(tt.input)
		if got != tt.want {
			t.Errorf("SeverityFromString(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
