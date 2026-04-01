package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/types"
)

func setupCommitSkill(t *testing.T, name string) VettedSkill {
	t.Helper()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`---
name: %s
description: "Use when testing the commit stage of the pipeline"
---

# %s

Instructions for testing.
`, name, name)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return VettedSkill{
		Candidate: SkillCandidate{
			Path:   skillDir,
			Source: "test-source",
			SHA:    "abc123",
			Origin: OriginLocal,
		},
		Skill: &types.Skill{
			Name:        name,
			Description: "Use when testing the commit stage of the pipeline",
		},
		ScanResult: &types.ScanResult{Score: 10, Risk: types.RiskLow},
		Selectable: true,
	}
}

func TestCommit_GlobalScope(t *testing.T) {
	coachDir := t.TempDir()
	agentDir := filepath.Join(t.TempDir(), "agent-skills")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	v1 := setupCommitSkill(t, "skill-one")
	v2 := setupCommitSkill(t, "skill-two")

	agents := []types.DetectedAgent{{
		Key:       "test-agent",
		Config:    types.AgentConfig{Name: "Test Agent"},
		Installed: true,
		SkillDir:  agentDir,
	}}

	results, err := Commit([]VettedSkill{v1, v2}, coachDir, InstallOptions{
		Scope:  "global",
		Agents: agents,
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Check skills exist in scope dir.
	globalSkills := filepath.Join(coachDir, "skills")
	for _, name := range []string{"skill-one", "skill-two"} {
		skillFile := filepath.Join(globalSkills, name, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			t.Errorf("expected %s to exist in global scope", skillFile)
		}
	}

	// Check skills distributed to agent.
	for _, name := range []string{"skill-one", "skill-two"} {
		agentSkill := filepath.Join(agentDir, name, "SKILL.md")
		if _, err := os.Stat(agentSkill); err != nil {
			t.Errorf("expected %s to exist in agent dir", agentSkill)
		}
	}

	// Check each result has the agent listed.
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("skill %s: unexpected error: %v", r.Name, r.Err)
		}
		if len(r.Agents) != 1 || r.Agents[0] != "Test Agent" {
			t.Errorf("skill %s: agents = %v, want [Test Agent]", r.Name, r.Agents)
		}
	}
}

func TestCommit_ProvenanceRecorded(t *testing.T) {
	coachDir := t.TempDir()
	v := setupCommitSkill(t, "tracked-skill")

	results, err := Commit([]VettedSkill{v}, coachDir, InstallOptions{
		Scope: "global",
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}

	provenance, err := registry.LoadProvenance(coachDir)
	if err != nil {
		t.Fatalf("LoadProvenance: %v", err)
	}
	if len(provenance.Skills) != 1 {
		t.Fatalf("got %d provenance records, want 1", len(provenance.Skills))
	}

	rec := provenance.Skills[0]
	if rec.Name != "tracked-skill" {
		t.Errorf("Name = %q, want %q", rec.Name, "tracked-skill")
	}
	if rec.Source != "test-source" {
		t.Errorf("Source = %q, want %q", rec.Source, "test-source")
	}
	if rec.CommitSHA != "abc123" {
		t.Errorf("CommitSHA = %q, want %q", rec.CommitSHA, "abc123")
	}
	if rec.ContentHash == "" {
		t.Error("ContentHash should be set")
	}
	if rec.RiskScore != 10 {
		t.Errorf("RiskScore = %d, want 10", rec.RiskScore)
	}
}

func TestCommit_CopyMode(t *testing.T) {
	coachDir := t.TempDir()
	v := setupCommitSkill(t, "copy-skill")

	_, err := Commit([]VettedSkill{v}, coachDir, InstallOptions{
		Scope: "global",
		Copy:  true,
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify the installed file is a regular file, not a symlink.
	installed := filepath.Join(coachDir, "skills", "copy-skill", "SKILL.md")
	info, err := os.Lstat(installed)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file in copy mode, got symlink")
	}
}

func TestCommit_SymlinkMode(t *testing.T) {
	coachDir := t.TempDir()
	v := setupCommitSkill(t, "link-skill")

	_, err := Commit([]VettedSkill{v}, coachDir, InstallOptions{
		Scope: "global",
		Copy:  false,
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	installed := filepath.Join(coachDir, "skills", "link-skill")
	info, err := os.Lstat(installed)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink in default mode, got regular file/dir")
	}
}

func TestCommit_MultipleAgents(t *testing.T) {
	coachDir := t.TempDir()
	agent1Dir := filepath.Join(t.TempDir(), "agent1")
	agent2Dir := filepath.Join(t.TempDir(), "agent2")
	for _, d := range []string{agent1Dir, agent2Dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	v := setupCommitSkill(t, "multi-agent-skill")

	agents := []types.DetectedAgent{
		{Key: "a1", Config: types.AgentConfig{Name: "Agent1"}, Installed: true, SkillDir: agent1Dir},
		{Key: "a2", Config: types.AgentConfig{Name: "Agent2"}, Installed: true, SkillDir: agent2Dir},
	}

	results, err := Commit([]VettedSkill{v}, coachDir, InstallOptions{
		Scope:  "global",
		Agents: agents,
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(results[0].Agents) != 2 {
		t.Errorf("distributed to %d agents, want 2", len(results[0].Agents))
	}

	// Both agent dirs should have the skill.
	for _, d := range []string{agent1Dir, agent2Dir} {
		p := filepath.Join(d, "multi-agent-skill", "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected skill in %s", p)
		}
	}
}

func TestCommit_SkipsUninstalledAgents(t *testing.T) {
	coachDir := t.TempDir()
	v := setupCommitSkill(t, "skip-agent-skill")

	agents := []types.DetectedAgent{
		{Key: "missing", Config: types.AgentConfig{Name: "Missing"}, Installed: false, SkillDir: "/nonexistent"},
	}

	results, err := Commit([]VettedSkill{v}, coachDir, InstallOptions{
		Scope:  "global",
		Agents: agents,
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(results[0].Agents) != 0 {
		t.Errorf("distributed to %d agents, want 0 (agent not installed)", len(results[0].Agents))
	}
}

func TestCommit_EmptySelected(t *testing.T) {
	coachDir := t.TempDir()

	results, err := Commit(nil, coachDir, InstallOptions{Scope: "global"})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
