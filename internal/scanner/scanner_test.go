package scanner

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dylan/coach/internal/rules"
	"github.com/dylan/coach/internal/skill"
	"github.com/dylan/coach/pkg"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}

func loadTestPatterns(t *testing.T) *pkg.PatternDatabase {
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
	result := ScanSkill(s, db)

	if result.Risk != pkg.RiskLow {
		t.Errorf("Risk = %v, want LOW. Findings: %v", result.Risk, result.Findings)
	}
}

func TestScanMaliciousSkill(t *testing.T) {
	s, err := skill.Parse(filepath.Join(testdataDir(), "malicious_skill"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	db := loadTestPatterns(t)
	result := ScanSkill(s, db)

	if result.Risk == pkg.RiskLow {
		t.Error("expected malicious skill to have risk > LOW")
	}

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
}

func TestScanMaliciousScripts(t *testing.T) {
	s, err := skill.Parse(filepath.Join(testdataDir(), "malicious_skill"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	db := loadTestPatterns(t)
	result := ScanSkill(s, db)

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
}

func TestScoreCalculation(t *testing.T) {
	findings := []pkg.Finding{
		{Severity: pkg.SeverityCritical, ID: "PI-001"},
		{Severity: pkg.SeverityCritical, ID: "PI-002"},
		{Severity: pkg.SeverityHigh, ID: "SC-001"},
	}
	score := CalculateScore(findings)

	if score != 75 {
		t.Errorf("score = %d, want 75", score)
	}
}

func TestScoreCapsAt100(t *testing.T) {
	findings := []pkg.Finding{
		{Severity: pkg.SeverityCritical, ID: "PI-001"},
		{Severity: pkg.SeverityCritical, ID: "PI-002"},
		{Severity: pkg.SeverityCritical, ID: "PI-003"},
		{Severity: pkg.SeverityCritical, ID: "PI-004"},
	}
	score := CalculateScore(findings)

	if score > 100 {
		t.Errorf("score = %d, should be capped at 100", score)
	}
}

func TestScoreDuplicatesCappedAt2x(t *testing.T) {
	findings := []pkg.Finding{
		{Severity: pkg.SeverityCritical, ID: "PI-001"},
		{Severity: pkg.SeverityCritical, ID: "PI-001"},
		{Severity: pkg.SeverityCritical, ID: "PI-001"},
	}
	score := CalculateScore(findings)

	if score != 60 {
		t.Errorf("score = %d, want 60 (capped at 2x for same pattern)", score)
	}
}

func TestQualityWarnings(t *testing.T) {
	s := &pkg.Skill{
		Name:        "test",
		Description: "Short",
		Body:        "Some body",
	}
	findings := CheckQuality(s)

	hasNoTools := false
	hasShortDesc := false
	for _, f := range findings {
		if f.ID == "QW-001" {
			hasNoTools = true
		}
		if f.ID == "QW-002" {
			hasShortDesc = true
		}
	}
	if !hasNoTools {
		t.Error("expected QW-001 (missing allowed-tools) finding")
	}
	if !hasShortDesc {
		t.Error("expected QW-002 (short description) finding")
	}
}
