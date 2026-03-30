package llm

import (
	"strings"
	"testing"
)

func TestBuildSingleShotArgs(t *testing.T) {
	args := BuildSingleShotArgs("system prompt here", "user instruction")
	expected := []string{"--print", "-p", "user instruction", "--system-prompt", "system prompt here"}
	if len(args) != len(expected) {
		t.Fatalf("args len = %d, want %d: %v", len(args), len(expected), args)
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildInteractiveArgs(t *testing.T) {
	args := BuildInteractiveArgs("system prompt here")
	expected := []string{"--system-prompt", "system prompt here"}
	if len(args) != len(expected) {
		t.Fatalf("args len = %d, want %d: %v", len(args), len(expected), args)
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildSystemPromptNewSkill(t *testing.T) {
	prompt := BuildSystemPrompt("", "")
	if prompt == "" {
		t.Fatal("system prompt should not be empty")
	}
	if !strings.Contains(prompt, "SKILL.md") {
		t.Error("system prompt should mention SKILL.md")
	}
	if !strings.Contains(prompt, "create a new skill from scratch") {
		t.Error("system prompt should instruct to create a new skill")
	}
	if strings.Contains(prompt, "Current Skill Content") {
		t.Error("system prompt should NOT include current skill section for new skills")
	}
}

func TestBuildSystemPromptExistingSkill(t *testing.T) {
	existing := "---\nname: test\ndescription: test skill\n---\nOld body."
	prompt := BuildSystemPrompt(existing, "")
	if !strings.Contains(prompt, "Old body") {
		t.Error("system prompt should include existing skill content")
	}
	if !strings.Contains(prompt, "Current Skill Content") {
		t.Error("system prompt should include current skill section")
	}
	if !strings.Contains(prompt, "refine and improve") {
		t.Error("system prompt should instruct to refine existing skill")
	}
}

func TestBuildSystemPromptReferenceOverride(t *testing.T) {
	override := "---\nname: custom-ref\ndescription: custom reference\n---\nCustom body."
	prompt := BuildSystemPrompt("", override)
	if !strings.Contains(prompt, "Custom body") {
		t.Error("system prompt should use reference override content")
	}
}

func TestBuildSystemPromptEmbeddedReference(t *testing.T) {
	prompt := BuildSystemPrompt("", "")
	if !strings.Contains(prompt, "skill-coach") {
		t.Error("system prompt should include embedded reference skill")
	}
}

func TestFindCLIMissing(t *testing.T) {
	_, err := FindCLI("nonexistent-cli-that-does-not-exist-xyz")
	if err == nil {
		t.Fatal("expected error for missing CLI")
	}
	if !strings.Contains(err.Error(), "CLI not found") {
		t.Errorf("error should mention 'CLI not found', got: %v", err)
	}
}
