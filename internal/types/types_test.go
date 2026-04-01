package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

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
		{"CRITICAL", SeverityCritical}, // case-insensitive
		{"High", SeverityHigh},         // mixed case
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

func TestInstalledSkill_ContentHash_YAMLRoundTrip(t *testing.T) {
	original := InstalledSkill{
		Name:        "test-skill",
		Source:      "owner/repo",
		CommitSHA:   "abc123",
		RiskScore:   15,
		ContentHash: "sha256:deadbeef1234",
		Agents:      []string{"claude-code"},
	}

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var loaded InstalledSkill
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded.ContentHash != original.ContentHash {
		t.Errorf("ContentHash: got %q, want %q", loaded.ContentHash, original.ContentHash)
	}
	if loaded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", loaded.Name, original.Name)
	}
}

func TestInstalledSkill_ContentHash_OmittedWhenEmpty(t *testing.T) {
	skill := InstalledSkill{
		Name:      "old-skill",
		Source:    "owner/repo",
		CommitSHA: "abc123",
		RiskScore: 10,
		Agents:    []string{"claude-code"},
	}

	data, err := yaml.Marshal(&skill)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// content_hash should not appear in output when empty
	if contains(string(data), "content_hash") {
		t.Errorf("expected content_hash to be omitted when empty, got:\n%s", data)
	}

	// Unmarshaling back should work with empty ContentHash
	var loaded InstalledSkill
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if loaded.ContentHash != "" {
		t.Errorf("ContentHash should be empty, got %q", loaded.ContentHash)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
