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
	DistributeTo   []string `yaml:"distribute_to,omitempty"`
	LLMCli         string   `yaml:"llm_cli,omitempty"`
	DefaultScope   string   `yaml:"default_scope,omitempty"`
}

var defaultConfig = Config{
	RulesSource:  "https://github.com/coach-dev/security-rules",
	LLMCli:       "claude",
	DefaultScope: "global",
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

// LoadFrom reads the Coach config from an explicit file path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
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

// SaveTo writes the config to an explicit file path.
func SaveTo(cfg Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads the Coach config from the given directory.
func Load(coachDir string) (*Config, error) {
	return LoadFrom(filepath.Join(coachDir, "config.yaml"))
}

// Save writes the config to disk.
func Save(coachDir string, cfg *Config) error {
	return SaveTo(*cfg, filepath.Join(coachDir, "config.yaml"))
}
