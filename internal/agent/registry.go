package agent

import (
	"github.com/dylan/coach/internal/rules"
	"github.com/dylan/coach/pkg"
)

// LoadRegistry loads the agent registry from embedded + optional override directory.
func LoadRegistry(overrideDir string) (*pkg.AgentRegistry, error) {
	return rules.LoadAgentRegistry(overrideDir)
}
