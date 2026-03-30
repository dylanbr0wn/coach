package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds Coach's global configuration.
type Config struct {
	RulesSource    string   `yaml:"rules_source"`
	TrustedSources []string `yaml:"trusted_sources,omitempty"`
	DefaultAgents  []string `yaml:"default_agents,omitempty"`
}

var defaultConfig = Config{
	RulesSource: "https://github.com/coach-dev/security-rules",
}

// DefaultCoachDir returns the default ~/.coach directory path.
func DefaultCoachDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".coach")
}

// EnsureCoachDir creates the ~/.coach directory and all subdirectories if they don't exist.
func EnsureCoachDir(coachDir string) error {
	subdirs := []string{"trust", "cache", "rules", "team"}
	for _, sub := range subdirs {
		path := filepath.Join(coachDir, sub)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", path, err)
		}
	}
	return nil
}

// Load reads the Coach config from the given directory.
func Load(coachDir string) (*Config, error) {
	configPath := filepath.Join(coachDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig
			return &cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := defaultConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk.
func Save(coachDir string, cfg *Config) error {
	configPath := filepath.Join(coachDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(configPath, data, 0o644)
}
