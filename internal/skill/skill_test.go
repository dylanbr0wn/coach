package skill

import (
	"os"
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

func TestListSkillDirs(t *testing.T) {
	dir := t.TempDir()

	// Create two skill directories with SKILL.md
	for _, name := range []string{"skill-a", "skill-b"} {
		skillDir := filepath.Join(dir, name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: test\n---\nBody"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a directory without SKILL.md (should be excluded)
	if err := os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	names := ListSkillDirs(dir)
	if len(names) != 2 {
		t.Fatalf("ListSkillDirs() returned %d names, want 2: %v", len(names), names)
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["skill-a"] || !nameSet["skill-b"] {
		t.Errorf("expected skill-a and skill-b, got %v", names)
	}
}

func TestListSkillDirs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := ListSkillDirs(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 names for empty dir, got %d", len(names))
	}
}

func TestListSkillDirs_NonexistentDir(t *testing.T) {
	names := ListSkillDirs("/nonexistent/path")
	if len(names) != 0 {
		t.Errorf("expected 0 names for nonexistent dir, got %d", len(names))
	}
}

func TestListSkillDirs_Symlinked(t *testing.T) {
	dir := t.TempDir()
	realDir := t.TempDir()

	// Create a real skill directory elsewhere
	skillDir := filepath.Join(realDir, "linked-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: linked-skill\ndescription: test\n---\nBody"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Symlink it into the listing directory
	if err := os.Symlink(skillDir, filepath.Join(dir, "linked-skill")); err != nil {
		t.Fatal(err)
	}

	names := ListSkillDirs(dir)
	if len(names) != 1 {
		t.Fatalf("ListSkillDirs() returned %d names, want 1: %v", len(names), names)
	}
	if names[0] != "linked-skill" {
		t.Errorf("expected 'linked-skill', got %q", names[0])
	}
}
