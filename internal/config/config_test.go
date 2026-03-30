package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCoachDir(t *testing.T) {
	tmpDir := t.TempDir()
	coachDir := filepath.Join(tmpDir, ".coach")

	err := EnsureCoachDir(coachDir)
	if err != nil {
		t.Fatalf("EnsureCoachDir() error: %v", err)
	}

	for _, sub := range []string{"trust", "cache", "rules", "team"} {
		path := filepath.Join(coachDir, sub)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected %s to exist", sub)
		}
	}
}

func TestLoadConfig_DefaultsOnMissing(t *testing.T) {
	tmpDir := t.TempDir()
	coachDir := filepath.Join(tmpDir, ".coach")
	if err := EnsureCoachDir(coachDir); err != nil {
		t.Fatalf("EnsureCoachDir() error: %v", err)
	}

	cfg, err := Load(coachDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.RulesSource == "" {
		t.Error("expected default RulesSource to be set")
	}
}

func TestCoachDir_ReturnsDefault(t *testing.T) {
	dir := DefaultCoachDir()
	if dir == "" {
		t.Error("DefaultCoachDir() returned empty string")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig
	if cfg.LLMCli != "claude" {
		t.Errorf("LLMCli default: got %q, want %q", cfg.LLMCli, "claude")
	}
	if cfg.DefaultScope != "global" {
		t.Errorf("DefaultScope default: got %q, want %q", cfg.DefaultScope, "global")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := Config{
		RulesSource:    "https://example.com/rules",
		TrustedSources: []string{"https://example.com"},
		DefaultAgents:  []string{"agent1"},
		DistributeTo:   []string{"team-a", "team-b"},
		LLMCli:         "llm",
		DefaultScope:   "local",
	}

	if err := SaveTo(original, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if loaded.RulesSource != original.RulesSource {
		t.Errorf("RulesSource: got %q, want %q", loaded.RulesSource, original.RulesSource)
	}
	if len(loaded.TrustedSources) != len(original.TrustedSources) || loaded.TrustedSources[0] != original.TrustedSources[0] {
		t.Errorf("TrustedSources: got %v, want %v", loaded.TrustedSources, original.TrustedSources)
	}
	if len(loaded.DefaultAgents) != len(original.DefaultAgents) || loaded.DefaultAgents[0] != original.DefaultAgents[0] {
		t.Errorf("DefaultAgents: got %v, want %v", loaded.DefaultAgents, original.DefaultAgents)
	}
	if len(loaded.DistributeTo) != len(original.DistributeTo) {
		t.Errorf("DistributeTo length: got %d, want %d", len(loaded.DistributeTo), len(original.DistributeTo))
	} else {
		for i, v := range original.DistributeTo {
			if loaded.DistributeTo[i] != v {
				t.Errorf("DistributeTo[%d]: got %q, want %q", i, loaded.DistributeTo[i], v)
			}
		}
	}
	if loaded.LLMCli != original.LLMCli {
		t.Errorf("LLMCli: got %q, want %q", loaded.LLMCli, original.LLMCli)
	}
	if loaded.DefaultScope != original.DefaultScope {
		t.Errorf("DefaultScope: got %q, want %q", loaded.DefaultScope, original.DefaultScope)
	}
}

func TestLoadFromNonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.yaml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom missing file: %v", err)
	}
	// Should return defaults
	if cfg.LLMCli != "claude" {
		t.Errorf("LLMCli default on missing file: got %q, want %q", cfg.LLMCli, "claude")
	}
	if cfg.DefaultScope != "global" {
		t.Errorf("DefaultScope default on missing file: got %q, want %q", cfg.DefaultScope, "global")
	}
}

func TestSaveLoadDelegation(t *testing.T) {
	dir := t.TempDir()

	original := Config{
		LLMCli:       "my-llm",
		DefaultScope: "local",
		DistributeTo: []string{"org/team"},
	}

	if err := Save(dir, &original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.LLMCli != original.LLMCli {
		t.Errorf("LLMCli: got %q, want %q", loaded.LLMCli, original.LLMCli)
	}
	if loaded.DefaultScope != original.DefaultScope {
		t.Errorf("DefaultScope: got %q, want %q", loaded.DefaultScope, original.DefaultScope)
	}
	if len(loaded.DistributeTo) != 1 || loaded.DistributeTo[0] != "org/team" {
		t.Errorf("DistributeTo: got %v, want [org/team]", loaded.DistributeTo)
	}
}

func TestSaveToCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{LLMCli: "claude", DefaultScope: "global"}
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
