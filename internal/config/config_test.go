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
