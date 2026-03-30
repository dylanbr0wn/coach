package agent

import (
	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/pkg"
)

// LoadRegistry loads the agent registry from embedded + optional override directory.
func LoadRegistry(overrideDir string) (*pkg.AgentRegistry, error) {
	return rules.LoadAgentRegistry(overrideDir)
}
