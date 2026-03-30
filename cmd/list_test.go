package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand_ShowsSkills(t *testing.T) {
	// Set up a temp home with one agent and two skills
	tmpHome := t.TempDir()
	skillsDir := filepath.Join(tmpHome, ".claude", "skills")

	for _, name := range []string{"alpha-skill", "beta-skill"} {
		dir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\nname: " + name + "\ndescription: A test skill\n---\nBody"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Set up coach dir with empty provenance
	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "", "table")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("alpha-skill")) {
		t.Errorf("expected output to contain 'alpha-skill', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("beta-skill")) {
		t.Errorf("expected output to contain 'beta-skill', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Claude Code")) {
		t.Errorf("expected output to contain 'Claude Code', got:\n%s", output)
	}
}

func TestListCommand_NoAgents(t *testing.T) {
	tmpHome := t.TempDir()
	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "", "table")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("No agents detected")) {
		t.Errorf("expected 'No agents detected' message, got:\n%s", output)
	}
}

func TestListCommand_AgentFilter(t *testing.T) {
	tmpHome := t.TempDir()
	skillsDir := filepath.Join(tmpHome, ".claude", "skills")
	dir := filepath.Join(skillsDir, "my-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: my-skill\ndescription: A test skill\n---\nBody"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "claude-code", "table")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("my-skill")) {
		t.Errorf("expected output to contain 'my-skill', got:\n%s", output)
	}
}

func TestListCommand_InvalidAgent(t *testing.T) {
	tmpHome := t.TempDir()
	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "nonexistent-agent", "table")
	if err == nil {
		t.Fatal("expected error for invalid agent, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-agent") {
		t.Errorf("error should mention the invalid agent name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("error should list available agents, got: %v", err)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	tmpHome := t.TempDir()
	skillsDir := filepath.Join(tmpHome, ".claude", "skills")
	dir := filepath.Join(skillsDir, "json-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: json-skill\ndescription: A JSON test skill\n---\nBody"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "claude-code", "json")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	var result []struct {
		Agent    string `json:"agent"`
		SkillDir string `json:"skill_dir"`
		Skills   []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Path        string `json:"path"`
			Vetted      bool   `json:"vetted"`
		} `json:"skills"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal error: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 agent group, got %d", len(result))
	}
	if len(result[0].Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result[0].Skills))
	}
	if result[0].Skills[0].Name != "json-skill" {
		t.Errorf("skill name = %q, want %q", result[0].Skills[0].Name, "json-skill")
	}
	if result[0].Skills[0].Description != "A JSON test skill" {
		t.Errorf("skill description = %q, want %q", result[0].Skills[0].Description, "A JSON test skill")
	}
}
