package distribute_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/distribute"
	"github.com/dylanbr0wn/coach/internal/types"
)

func makeAgent(t *testing.T, name string, installed bool) (agent types.DetectedAgent, skillDir string) {
	t.Helper()
	skillDir = t.TempDir()
	return types.DetectedAgent{
		Config: types.AgentConfig{
			Name: name,
		},
		Installed: installed,
		SkillDir:  skillDir,
	}, skillDir
}

func TestDistributeCreatesSymlink(t *testing.T) {
	source := t.TempDir()
	agent, agentSkillDir := makeAgent(t, "claude", true)

	results, err := distribute.Distribute(source, "my-skill", []types.DetectedAgent{agent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Status != distribute.StatusCreated {
		t.Errorf("expected StatusCreated, got %v", r.Status)
	}
	if r.Agent != "claude" {
		t.Errorf("expected agent 'claude', got %q", r.Agent)
	}

	linkPath := filepath.Join(agentSkillDir, "my-skill")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if target != source {
		t.Errorf("symlink target: got %q, want %q", target, source)
	}
}

func TestDistributeAlreadyLinked(t *testing.T) {
	source := t.TempDir()
	agent, agentSkillDir := makeAgent(t, "claude", true)

	// Pre-create correct symlink
	linkPath := filepath.Join(agentSkillDir, "my-skill")
	if err := os.Symlink(source, linkPath); err != nil {
		t.Fatalf("setup: %v", err)
	}

	results, err := distribute.Distribute(source, "my-skill", []types.DetectedAgent{agent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != distribute.StatusUpToDate {
		t.Errorf("expected StatusUpToDate, got %v", results[0].Status)
	}
}

func TestDistributeUpdatesStaleSymlink(t *testing.T) {
	source := t.TempDir()
	otherSource := t.TempDir()
	agent, agentSkillDir := makeAgent(t, "claude", true)

	// Pre-create symlink pointing elsewhere
	linkPath := filepath.Join(agentSkillDir, "my-skill")
	if err := os.Symlink(otherSource, linkPath); err != nil {
		t.Fatalf("setup: %v", err)
	}

	results, err := distribute.Distribute(source, "my-skill", []types.DetectedAgent{agent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != distribute.StatusUpdated {
		t.Errorf("expected StatusUpdated, got %v", results[0].Status)
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("symlink missing after update: %v", err)
	}
	if target != source {
		t.Errorf("symlink target after update: got %q, want %q", target, source)
	}
}

func TestDistributeSkipsUninstalledAgents(t *testing.T) {
	source := t.TempDir()
	agent, _ := makeAgent(t, "cursor", false)
	agent.Installed = false

	results, err := distribute.Distribute(source, "my-skill", []types.DetectedAgent{agent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != distribute.StatusSkipped {
		t.Errorf("expected StatusSkipped, got %v", results[0].Status)
	}
}

func TestFilterAgentsByNames(t *testing.T) {
	agents := []types.DetectedAgent{
		{Config: types.AgentConfig{Name: "claude"}, Installed: true},
		{Config: types.AgentConfig{Name: "cursor"}, Installed: true},
		{Config: types.AgentConfig{Name: "windsurf"}, Installed: false},
	}

	filtered := distribute.FilterAgentsByNames(agents, []string{"claude", "windsurf"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered agents, got %d", len(filtered))
	}

	names := map[string]bool{}
	for _, a := range filtered {
		names[a.Config.Name] = true
	}
	if !names["claude"] || !names["windsurf"] {
		t.Errorf("wrong agents filtered: got %v", names)
	}
	if names["cursor"] {
		t.Errorf("cursor should have been excluded")
	}
}

func TestFilterAgentsByNames_MatchesKey(t *testing.T) {
	agents := []types.DetectedAgent{
		{Key: "claude-code", Config: types.AgentConfig{Name: "Claude Code"}, Installed: true},
		{Key: "cursor", Config: types.AgentConfig{Name: "Cursor"}, Installed: true},
	}

	filtered := distribute.FilterAgentsByNames(agents, []string{"claude-code"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered agent, got %d", len(filtered))
	}
	if filtered[0].Key != "claude-code" {
		t.Errorf("expected claude-code, got %q", filtered[0].Key)
	}
}

func TestStatusString(t *testing.T) {
	cases := []struct {
		s    distribute.Status
		want string
	}{
		{distribute.StatusCreated, "created"},
		{distribute.StatusUpdated, "updated"},
		{distribute.StatusUpToDate, "up-to-date"},
		{distribute.StatusSkipped, "skipped"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("Status(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}
