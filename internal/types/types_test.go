package types

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
