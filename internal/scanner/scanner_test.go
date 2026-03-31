package scanner

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/types"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}

func loadTestPatterns(t *testing.T) *types.PatternDatabase {
	t.Helper()
	db, err := rules.LoadPatterns("")
	if err != nil {
		t.Fatalf("LoadPatterns() error: %v", err)
	}
	return db
}

func TestScanValidSkill(t *testing.T) {
	s, err := skill.Parse(filepath.Join(testdataDir(), "valid_skill"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	db := loadTestPatterns(t)
	result, err := ScanSkill(s, db)
	if err != nil {
		t.Fatalf("ScanSkill error: %v", err)
	}

	if result.Risk != types.RiskLow {
		t.Errorf("Risk = %v, want LOW. Findings: %v", result.Risk, result.Findings)
	}
}

func TestScanMaliciousSkill(t *testing.T) {
	s, err := skill.Parse(filepath.Join(testdataDir(), "malicious_skill"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	db := loadTestPatterns(t)
	result, err := ScanSkill(s, db)
	if err != nil {
		t.Fatalf("ScanSkill error: %v", err)
	}

	if result.Risk == types.RiskLow {
		t.Error("expected malicious skill to have risk > LOW")
	}

	t.Run("has prompt-injection finding", func(t *testing.T) {
		hasInjection := false
		for _, f := range result.Findings {
			if f.Category == "prompt-injection" {
				hasInjection = true
				break
			}
		}
		if !hasInjection {
			t.Error("expected at least one prompt-injection finding")
		}
	})

	t.Run("has script-danger finding", func(t *testing.T) {
		hasScriptDanger := false
		for _, f := range result.Findings {
			if f.Category == "script-danger" {
				hasScriptDanger = true
				break
			}
		}
		if !hasScriptDanger {
			t.Error("expected at least one script-danger finding")
		}
	})
}

func TestScoreCalculation(t *testing.T) {
	findings := []types.Finding{
		{Severity: types.SeverityCritical, ID: "PI-001"},
		{Severity: types.SeverityCritical, ID: "PI-002"},
		{Severity: types.SeverityHigh, ID: "SC-001"},
	}
	score := CalculateScore(findings)

	if score != 75 {
		t.Errorf("CalculateScore(%v) = %d, want 75", findings, score)
	}
}

func TestScoreCapsAt100(t *testing.T) {
	findings := []types.Finding{
		{Severity: types.SeverityCritical, ID: "PI-001"},
		{Severity: types.SeverityCritical, ID: "PI-002"},
		{Severity: types.SeverityCritical, ID: "PI-003"},
		{Severity: types.SeverityCritical, ID: "PI-004"},
	}
	score := CalculateScore(findings)

	if score > 100 {
		t.Errorf("CalculateScore(%v) = %d, want <= 100", findings, score)
	}
}

func TestScoreDuplicatesCappedAt2x(t *testing.T) {
	findings := []types.Finding{
		{Severity: types.SeverityCritical, ID: "PI-001"},
		{Severity: types.SeverityCritical, ID: "PI-001"},
		{Severity: types.SeverityCritical, ID: "PI-001"},
	}
	score := CalculateScore(findings)

	if score != 60 {
		t.Errorf("CalculateScore(%v) = %d, want 60 (capped at 2x for same pattern)", findings, score)
	}
}

func TestQualityWarnings(t *testing.T) {
	s := &types.Skill{
		Name:        "test",
		Description: "Short",
		Body:        "Some body",
	}
	findings := CheckQuality(s)

	tests := []struct {
		name string
		id   string
	}{
		{"missing allowed-tools", "QW-001"},
		{"short description", "QW-002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, f := range findings {
				if f.ID == tt.id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected finding %s (%s)", tt.id, tt.name)
			}
		})
	}
}
