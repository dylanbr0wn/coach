package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgents_FindsClaudeCode(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents, err := DetectAgentsInHome(tmpHome)
	if err != nil {
		t.Fatalf("DetectAgentsInHome() error: %v", err)
	}

	found := false
	for _, a := range agents {
		if a.Config.Name == "Claude Code" && a.Installed {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Claude Code to be detected")
	}
}

func TestDetectAgents_SkipsMissing(t *testing.T) {
	tmpHome := t.TempDir()

	agents, err := DetectAgentsInHome(tmpHome)
	if err != nil {
		t.Fatalf("DetectAgentsInHome() error: %v", err)
	}

	for _, a := range agents {
		if a.Installed {
			t.Errorf("agent %q should not be detected in empty home", a.Config.Name)
		}
	}
}

func TestDetectAgents_HasKey(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents, err := DetectAgentsInHome(tmpHome)
	if err != nil {
		t.Fatalf("DetectAgentsInHome() error: %v", err)
	}

	for _, a := range agents {
		if a.Key == "" {
			t.Errorf("agent %q has empty Key", a.Config.Name)
		}
	}

	// Check specific key mapping
	for _, a := range agents {
		if a.Config.Name == "Claude Code" && a.Key != "claude-code" {
			t.Errorf("Claude Code Key = %q, want %q", a.Key, "claude-code")
		}
	}
}

func TestResolveHomePath(t *testing.T) {
	result := resolveHomePath("~/.claude/skills/", "/Users/test")
	expected := "/Users/test/.claude/skills/"
	if result != expected {
		t.Errorf("resolveHomePath = %q, want %q", result, expected)
	}
}
