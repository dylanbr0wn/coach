package rules

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan/coach/pkg"
	"gopkg.in/yaml.v3"
)

// SeverityFromString converts a string severity to the Severity type.
func SeverityFromString(s string) pkg.Severity {
	switch s {
	case "critical":
		return pkg.SeverityCritical
	case "high":
		return pkg.SeverityHigh
	case "medium":
		return pkg.SeverityMedium
	case "warning":
		return pkg.SeverityWarning
	case "info":
		return pkg.SeverityInfo
	default:
		return pkg.SeverityInfo
	}
}

// LoadPatterns loads the pattern database. If overrideDir is non-empty,
// it merges patterns from that directory on top of the embedded defaults.
// Remote patterns take priority (matched by ID).
func LoadPatterns(overrideDir string) (*pkg.PatternDatabase, error) {
	var db pkg.PatternDatabase
	if err := yaml.Unmarshal(embeddedPatterns, &db); err != nil {
		return nil, fmt.Errorf("parsing embedded patterns: %w", err)
	}

	if overrideDir == "" {
		return &db, nil
	}

	overridePath := filepath.Join(overrideDir, "patterns.yaml")
	data, err := os.ReadFile(overridePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &db, nil
		}
		return nil, fmt.Errorf("reading override patterns: %w", err)
	}

	var override pkg.PatternDatabase
	if err := yaml.Unmarshal(data, &override); err != nil {
		return nil, fmt.Errorf("parsing override patterns: %w", err)
	}

	db = mergePatterns(db, override)
	return &db, nil
}

// LoadAgentRegistry loads the agent registry. If overrideDir is non-empty,
// it merges agents from that directory on top of the embedded defaults.
func LoadAgentRegistry(overrideDir string) (*pkg.AgentRegistry, error) {
	var reg pkg.AgentRegistry
	if err := yaml.Unmarshal(embeddedAgents, &reg); err != nil {
		return nil, fmt.Errorf("parsing embedded agents: %w", err)
	}

	if overrideDir == "" {
		return &reg, nil
	}

	overridePath := filepath.Join(overrideDir, "agents.yaml")
	data, err := os.ReadFile(overridePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &reg, nil
		}
		return nil, fmt.Errorf("reading override agents: %w", err)
	}

	var override pkg.AgentRegistry
	if err := yaml.Unmarshal(data, &override); err != nil {
		return nil, fmt.Errorf("parsing override agents: %w", err)
	}

	for k, v := range override.Agents {
		reg.Agents[k] = v
	}
	return &reg, nil
}

// mergePatterns merges override patterns into base. Override patterns
// with matching IDs replace base patterns. New override patterns are appended.
func mergePatterns(base, override pkg.PatternDatabase) pkg.PatternDatabase {
	idIndex := make(map[string]int, len(base.Patterns))
	for i, p := range base.Patterns {
		idIndex[p.ID] = i
	}

	for _, op := range override.Patterns {
		if idx, exists := idIndex[op.ID]; exists {
			base.Patterns[idx] = op
		} else {
			base.Patterns = append(base.Patterns, op)
		}
	}
	return base
}
