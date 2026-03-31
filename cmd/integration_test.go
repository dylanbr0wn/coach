package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/distribute"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/types"
)

func TestFullWorkflow(t *testing.T) {
	// Setup: fake home dir with coach structure.
	// Use separate temp dirs so the global skills dir isn't found by the local
	// walk-up search (which looks for .coach/skills relative to WorkDir).
	homeDir := t.TempDir()
	globalDir := t.TempDir()
	globalSkillsDir := filepath.Join(globalDir, "skills")
	agentSkillDir := filepath.Join(homeDir, ".claude", "skills")
	if err := os.MkdirAll(globalSkillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(agentSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(homeDir, ".coach"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Step 1: Create a skill directory with valid SKILL.md
	skillName := "test-reviewer"
	skillDir := filepath.Join(globalSkillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `---
name: test-reviewer
description: Reviews test quality and coverage
license: MIT
allowed-tools:
  - Read
  - Grep
---

# Test Reviewer

## When to Use

Use when the user asks to review tests or check test coverage.

## Instructions

1. Read the test files in the project
2. Check for common testing anti-patterns
3. Suggest improvements

## Constraints

- Do not modify test files without asking
- Do not delete existing tests
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Step 2: Resolve the skill
	r := resolve.Resolver{GlobalSkillsDir: globalSkillsDir, WorkDir: homeDir}
	result, err := r.Resolve(skillName, resolve.ScopeAny)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if result.Scope != resolve.ScopeGlobal {
		t.Errorf("scope = %v, want ScopeGlobal", result.Scope)
	}

	// Step 3: Parse and validate
	s, err := skill.Parse(result.Dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	errors := skill.Validate(s)
	if len(errors) > 0 {
		t.Errorf("Validate returned errors: %v", errors)
	}

	// Step 4: Distribute
	agents := []types.DetectedAgent{
		{
			Config:    types.AgentConfig{Name: "claude-code", SkillDir: agentSkillDir + "/"},
			Installed: true,
			SkillDir:  agentSkillDir,
		},
	}

	results, err := distribute.Distribute(result.Dir, skillName, agents)
	if err != nil {
		t.Fatalf("Distribute failed: %v", err)
	}
	if results[0].Status != distribute.StatusCreated {
		t.Errorf("status = %v, want StatusCreated", results[0].Status)
	}

	// Verify symlink exists and points to right place
	linkPath := filepath.Join(agentSkillDir, skillName)
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if target != result.Dir {
		t.Errorf("symlink target = %q, want %q", target, result.Dir)
	}

	// Step 5: Config round-trip
	configPath := filepath.Join(homeDir, ".coach", "config.yaml")
	cfg := config.Config{
		DistributeTo: []string{"claude-code"},
		LLMCli:       "claude",
		DefaultScope: "global",
	}
	if err := config.SaveTo(&cfg, configPath); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}
	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if loaded.LLMCli != "claude" {
		t.Errorf("LLMCli = %q, want %q", loaded.LLMCli, "claude")
	}
}
