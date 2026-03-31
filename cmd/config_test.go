package cmd

import (
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/config"
)

func TestConfigSetAndGet(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	if err := setConfigValue(configPath, "llm-cli", "codex"); err != nil {
		t.Fatalf("setConfigValue failed: %v", err)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.LLMCli != "codex" {
		t.Errorf("LLMCli = %q, want %q", cfg.LLMCli, "codex")
	}

	if err := setConfigValue(configPath, "distribute-to", "claude,cursor"); err != nil {
		t.Fatalf("setConfigValue failed: %v", err)
	}

	cfg, err = config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if len(cfg.DistributeTo) != 2 || cfg.DistributeTo[0] != "claude" {
		t.Errorf("DistributeTo = %v, want [claude cursor]", cfg.DistributeTo)
	}
}

func TestConfigSetInvalidKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	err := setConfigValue(configPath, "invalid-key", "value")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestConfigSetInvalidScope(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	err := setConfigValue(configPath, "default-scope", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid scope value")
	}
}

func TestConfigGetValue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	cfg := config.Config{LLMCli: "claude", DistributeTo: []string{"claude", "cursor"}}
	if err := config.SaveTo(&cfg, configPath); err != nil {
		t.Fatal(err)
	}

	val, err := getConfigValue(configPath, "llm-cli")
	if err != nil {
		t.Fatalf("getConfigValue failed: %v", err)
	}
	if val != "claude" {
		t.Errorf("got %q, want %q", val, "claude")
	}

	val, err = getConfigValue(configPath, "distribute-to")
	if err != nil {
		t.Fatalf("getConfigValue failed: %v", err)
	}
	if val != "claude,cursor" {
		t.Errorf("got %q, want %q", val, "claude,cursor")
	}
}

func TestConfigGetInvalidKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	_, err := getConfigValue(configPath, "bad-key")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
