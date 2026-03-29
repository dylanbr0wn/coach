package skill

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}

func TestParseValidSkill(t *testing.T) {
	s, err := Parse(filepath.Join(testdataDir(), "valid_skill"))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if s.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "test-skill")
	}
	if s.Description != "A valid test skill for unit testing Coach" {
		t.Errorf("Description = %q, want %q", s.Description, "A valid test skill for unit testing Coach")
	}
	if s.License != "MIT" {
		t.Errorf("License = %q, want %q", s.License, "MIT")
	}
	if len(s.AllowedTools) != 2 {
		t.Errorf("AllowedTools length = %d, want 2", len(s.AllowedTools))
	}
	if s.Body == "" {
		t.Error("Body should not be empty")
	}
}

func TestParseInvalidSkill_MissingName(t *testing.T) {
	_, err := Parse(filepath.Join(testdataDir(), "invalid_skill"))
	if err == nil {
		t.Fatal("expected error for skill missing name, got nil")
	}
}

func TestParseNonexistentPath(t *testing.T) {
	_, err := Parse("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestValidate(t *testing.T) {
	s, _ := Parse(filepath.Join(testdataDir(), "valid_skill"))
	errs := Validate(s)
	if len(errs) != 0 {
		t.Errorf("expected 0 validation errors, got %d: %v", len(errs), errs)
	}
}
