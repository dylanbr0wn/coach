package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/spf13/cobra"
)

var validConfigKeys = map[string]bool{
	"distribute-to": true,
	"llm-cli":       true,
	"default-scope": true,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Get and set Coach configuration values",
	Long:  "Manage Coach configuration. Use 'config set' to update values and 'config get' to read them.",
	Example: `  coach config set distribute-to claude,cursor    # Set distribution targets
  coach config set llm-cli claude                  # Set default LLM CLI
  coach config get distribute-to                   # Show current value`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(config.DefaultCoachDir(), "config.yaml")
		return setConfigValue(configPath, args[0], args[1])
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(config.DefaultCoachDir(), "config.yaml")
		val, err := getConfigValue(configPath, args[0])
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	rootCmd.AddCommand(configCmd)
}

func setConfigValue(configPath, key, value string) error {
	if !validConfigKeys[key] {
		return fmt.Errorf("unknown config key %q; valid keys: distribute-to, llm-cli, default-scope", key)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	switch key {
	case "distribute-to":
		parts := strings.Split(value, ",")
		targets := make([]string, 0, len(parts))
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				targets = append(targets, t)
			}
		}
		cfg.DistributeTo = targets
	case "llm-cli":
		cfg.LLMCli = value
	case "default-scope":
		if value != "global" && value != "local" {
			return fmt.Errorf("invalid value %q for default-scope; must be \"global\" or \"local\"", value)
		}
		cfg.DefaultScope = value
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return config.SaveTo(*cfg, configPath)
}

func getConfigValue(configPath, key string) (string, error) {
	if !validConfigKeys[key] {
		return "", fmt.Errorf("unknown config key %q; valid keys: distribute-to, llm-cli, default-scope", key)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	switch key {
	case "distribute-to":
		return strings.Join(cfg.DistributeTo, ","), nil
	case "llm-cli":
		return cfg.LLMCli, nil
	case "default-scope":
		return cfg.DefaultScope, nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}
