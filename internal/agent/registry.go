package agent

import (
	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/internal/types"
)

// LoadRegistry loads the agent registry from embedded + optional override directory.
func LoadRegistry(overrideDir string) (*types.AgentRegistry, error) {
	return rules.LoadAgentRegistry(overrideDir)
}
