package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/resolve"
)

// writeSkill creates a skill directory with SKILL.md under dir/name.
func writeSkill(t *testing.T, dir, name string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: test skill\n---\nBody here.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return skillDir
}

// makeLocalSkillsDir creates <root>/.coach/skills and returns its path.
func makeLocalSkillsDir(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, ".coach", "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll local skills dir: %v", err)
	}
	return dir
}

func TestResolveLocal(t *testing.T) {
	tmp := t.TempDir()
	localSkills := makeLocalSkillsDir(t, tmp)
	writeSkill(t, localSkills, "my-skill")

	// WorkDir is a subdir of the project root to simulate walking up.
	workDir := filepath.Join(tmp, "subdir", "deep")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: t.TempDir(), // empty global
		WorkDir:         workDir,
	}

	result, err := r.Resolve("my-skill", resolve.ScopeLocal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", result.Name, "my-skill")
	}
	if result.Scope != resolve.ScopeLocal {
		t.Errorf("Scope = %v, want ScopeLocal", result.Scope)
	}
	if result.Path != filepath.Join(localSkills, "my-skill", "SKILL.md") {
		t.Errorf("Path = %q, unexpected", result.Path)
	}
}

func TestResolveGlobal(t *testing.T) {
	tmp := t.TempDir()
	globalSkills := t.TempDir()
	writeSkill(t, globalSkills, "global-skill")

	// No local skills dir at all.
	workDir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         workDir,
	}

	result, err := r.Resolve("global-skill", resolve.ScopeGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Scope != resolve.ScopeGlobal {
		t.Errorf("Scope = %v, want ScopeGlobal", result.Scope)
	}
	if result.Name != "global-skill" {
		t.Errorf("Name = %q, want %q", result.Name, "global-skill")
	}
}

func TestResolveLocalOverridesGlobal(t *testing.T) {
	tmp := t.TempDir()
	localSkills := makeLocalSkillsDir(t, tmp)
	globalSkills := t.TempDir()

	// Same name in both.
	writeSkill(t, localSkills, "shared-skill")
	writeSkill(t, globalSkills, "shared-skill")

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         tmp,
	}

	result, err := r.Resolve("shared-skill", resolve.ScopeAny)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Scope != resolve.ScopeLocal {
		t.Errorf("Scope = %v, want ScopeLocal (local should win)", result.Scope)
	}
}

func TestResolveNotFound(t *testing.T) {
	tmp := t.TempDir()
	r := resolve.Resolver{
		GlobalSkillsDir: t.TempDir(),
		WorkDir:         tmp,
	}

	_, err := r.Resolve("missing-skill", resolve.ScopeAny)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Error should mention the skill name.
	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Error("expected non-empty error message")
	}
}

func TestResolveForcedScope(t *testing.T) {
	tmp := t.TempDir()
	globalSkills := t.TempDir()
	writeSkill(t, globalSkills, "global-only")

	// No local skills.
	workDir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         workDir,
	}

	// Forced ScopeLocal should fail.
	_, err := r.Resolve("global-only", resolve.ScopeLocal)
	if err == nil {
		t.Error("expected error when forcing ScopeLocal for global-only skill")
	}

	// Forced ScopeGlobal should succeed.
	result, err := r.Resolve("global-only", resolve.ScopeGlobal)
	if err != nil {
		t.Fatalf("unexpected error for ScopeGlobal: %v", err)
	}
	if result.Scope != resolve.ScopeGlobal {
		t.Errorf("Scope = %v, want ScopeGlobal", result.Scope)
	}
}

func TestListSkills(t *testing.T) {
	tmp := t.TempDir()
	localSkills := makeLocalSkillsDir(t, tmp)
	globalSkills := t.TempDir()

	writeSkill(t, localSkills, "local-only")
	writeSkill(t, localSkills, "shared")
	writeSkill(t, globalSkills, "shared") // shadowed by local
	writeSkill(t, globalSkills, "global-only")

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         tmp,
	}

	results, err := r.List(resolve.ScopeAny)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Build a map for easy checking.
	byName := make(map[string]resolve.Result)
	for _, res := range results {
		byName[res.Name] = res
	}

	// All three unique names should appear.
	if len(byName) != 3 {
		t.Errorf("got %d unique skills, want 3: %v", len(byName), byName)
	}

	// local-only is local.
	if s, ok := byName["local-only"]; !ok {
		t.Error("missing local-only")
	} else if s.Scope != resolve.ScopeLocal {
		t.Errorf("local-only scope = %v, want ScopeLocal", s.Scope)
	}

	// global-only is global.
	if s, ok := byName["global-only"]; !ok {
		t.Error("missing global-only")
	} else if s.Scope != resolve.ScopeGlobal {
		t.Errorf("global-only scope = %v, want ScopeGlobal", s.Scope)
	}

	// shared is local (shadowed).
	if s, ok := byName["shared"]; !ok {
		t.Error("missing shared")
	} else if s.Scope != resolve.ScopeLocal {
		t.Errorf("shared scope = %v, want ScopeLocal", s.Scope)
	}
}

func TestListSkillsLocalOnly(t *testing.T) {
	tmp := t.TempDir()
	localSkills := makeLocalSkillsDir(t, tmp)
	globalSkills := t.TempDir()

	writeSkill(t, localSkills, "local-skill")
	writeSkill(t, globalSkills, "global-skill")

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         tmp,
	}

	results, err := r.List(resolve.ScopeLocal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "local-skill" {
		t.Errorf("got %q, want local-skill", results[0].Name)
	}
}

func TestListSkillsGlobalOnly(t *testing.T) {
	tmp := t.TempDir()
	globalSkills := t.TempDir()

	writeSkill(t, globalSkills, "global-skill")
	// No local skills dir.

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         tmp,
	}

	results, err := r.List(resolve.ScopeGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "global-skill" {
		t.Errorf("got %q, want global-skill", results[0].Name)
	}
}

func TestTargetDir(t *testing.T) {
	tmp := t.TempDir()
	globalSkills := t.TempDir()

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         tmp,
	}

	localTarget := r.TargetDir("new-skill", resolve.ScopeLocal)
	expectedLocal := filepath.Join(tmp, ".coach", "skills", "new-skill")
	if localTarget != expectedLocal {
		t.Errorf("TargetDir(local) = %q, want %q", localTarget, expectedLocal)
	}

	globalTarget := r.TargetDir("new-skill", resolve.ScopeGlobal)
	expectedGlobal := filepath.Join(globalSkills, "new-skill")
	if globalTarget != expectedGlobal {
		t.Errorf("TargetDir(global) = %q, want %q", globalTarget, expectedGlobal)
	}
}

func TestListSkills_Symlinked(t *testing.T) {
	tmp := t.TempDir()
	globalSkills := t.TempDir()
	realDir := t.TempDir()

	// Create a real skill directory elsewhere
	skillDir := filepath.Join(realDir, "linked-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: linked-skill\ndescription: test skill\n---\nBody here.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Symlink into global skills dir
	if err := os.Symlink(skillDir, filepath.Join(globalSkills, "linked-skill")); err != nil {
		t.Fatal(err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: globalSkills,
		WorkDir:         tmp,
	}

	results, err := r.List(resolve.ScopeGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "linked-skill" {
		t.Errorf("got %q, want linked-skill", results[0].Name)
	}
}
