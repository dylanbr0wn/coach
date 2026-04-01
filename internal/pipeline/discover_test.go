package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/types"
)

func writeSkillMD(t *testing.T, dir, name, content string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

const validSkill = `---
name: %s
description: A test skill
---

# %s

Instructions here.
`

func TestDiscover_LocalMultiSkill(t *testing.T) {
	dir := t.TempDir()
	writeSkillMD(t, dir, "skill-a", fmt.Sprintf(validSkill, "skill-a", "skill-a"))
	writeSkillMD(t, dir, "skill-b", fmt.Sprintf(validSkill, "skill-b", "skill-b"))
	writeSkillMD(t, dir, "skill-c", fmt.Sprintf(validSkill, "skill-c", "skill-c"))

	src := &registry.Source{
		Type: registry.SourceLocal,
		Raw:  dir,
		Path: dir,
	}

	candidates, err := Discover(src, false, nil, nil)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("got %d candidates, want 3", len(candidates))
	}
	for _, c := range candidates {
		if c.Origin != OriginLocal {
			t.Errorf("candidate %s: origin = %v, want OriginLocal", c.Path, c.Origin)
		}
		if c.SHA != "local" {
			t.Errorf("candidate %s: SHA = %q, want \"local\"", c.Path, c.SHA)
		}
	}
}

func TestDiscover_LocalFlatLayout(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: flat-skill
description: A flat layout skill
---

Body.
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	src := &registry.Source{
		Type: registry.SourceLocal,
		Raw:  dir,
		Path: dir,
	}

	candidates, err := Discover(src, false, nil, nil)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
}

func TestDiscover_LocalNoSkills(t *testing.T) {
	dir := t.TempDir()

	src := &registry.Source{
		Type: registry.SourceLocal,
		Raw:  dir,
		Path: dir,
	}

	_, err := Discover(src, false, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestDiscover_NilSourceWithoutInstalled(t *testing.T) {
	_, err := Discover(nil, false, nil, nil)
	if err == nil {
		t.Fatal("expected error when src is nil and installed is false")
	}
}

func TestDiscover_InstalledUntracked(t *testing.T) {
	agentDir := t.TempDir()
	writeSkillMD(t, agentDir, "untracked-skill", `---
name: untracked-skill
description: Not in provenance
---

Body.
`)

	agents := []types.DetectedAgent{{
		Key:       "test-agent",
		Config:    types.AgentConfig{Name: "Test Agent"},
		Installed: true,
		SkillDir:  agentDir,
	}}
	provenance := &registry.InstalledSkills{}

	candidates, err := Discover(nil, true, agents, provenance)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].Origin != OriginInstalledUntracked {
		t.Errorf("origin = %v, want OriginInstalledUntracked", candidates[0].Origin)
	}
}

func TestDiscover_InstalledModified(t *testing.T) {
	agentDir := t.TempDir()
	content := `---
name: modified-skill
description: Was changed after install
---

New body.
`
	writeSkillMD(t, agentDir, "modified-skill", content)

	agents := []types.DetectedAgent{{
		Key:       "test-agent",
		Config:    types.AgentConfig{Name: "Test Agent"},
		Installed: true,
		SkillDir:  agentDir,
	}}
	provenance := &registry.InstalledSkills{
		Skills: []types.InstalledSkill{{
			Name:        "modified-skill",
			ContentHash: "oldhashvalue",
			Agents:      []string{"test-agent"},
		}},
	}

	candidates, err := Discover(nil, true, agents, provenance)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].Origin != OriginInstalledModified {
		t.Errorf("origin = %v, want OriginInstalledModified", candidates[0].Origin)
	}
}

func TestDiscover_InstalledMatchingHash(t *testing.T) {
	agentDir := t.TempDir()
	content := `---
name: unchanged-skill
description: Same as installed
---

Body.
`
	writeSkillMD(t, agentDir, "unchanged-skill", content)

	// Compute the actual hash of the file we wrote.
	hash, err := ContentHash(filepath.Join(agentDir, "unchanged-skill", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}

	agents := []types.DetectedAgent{{
		Key:       "test-agent",
		Config:    types.AgentConfig{Name: "Test Agent"},
		Installed: true,
		SkillDir:  agentDir,
	}}
	provenance := &registry.InstalledSkills{
		Skills: []types.InstalledSkill{{
			Name:        "unchanged-skill",
			ContentHash: hash,
			Agents:      []string{"test-agent"},
		}},
	}

	candidates, err := Discover(nil, true, agents, provenance)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("got %d candidates, want 0 (unchanged skill should be skipped)", len(candidates))
	}
}

func TestDiscover_InstalledDeduplicatesAcrossAgents(t *testing.T) {
	// Same skill in two agent dirs should only appear once.
	agent1Dir := t.TempDir()
	agent2Dir := t.TempDir()
	content := `---
name: shared-skill
description: In both agents
---

Body.
`
	writeSkillMD(t, agent1Dir, "shared-skill", content)
	writeSkillMD(t, agent2Dir, "shared-skill", content)

	agents := []types.DetectedAgent{
		{Key: "agent-1", Config: types.AgentConfig{Name: "Agent 1"}, Installed: true, SkillDir: agent1Dir},
		{Key: "agent-2", Config: types.AgentConfig{Name: "Agent 2"}, Installed: true, SkillDir: agent2Dir},
	}
	provenance := &registry.InstalledSkills{}

	candidates, err := Discover(nil, true, agents, provenance)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1 (should deduplicate)", len(candidates))
	}
}

func TestContentHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := ContentHash(path)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	// SHA-256 of "hello" is known.
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if hash != want {
		t.Errorf("hash = %q, want %q", hash, want)
	}
}

func TestContentHash_MissingFile(t *testing.T) {
	_, err := ContentHash("/nonexistent/file.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
