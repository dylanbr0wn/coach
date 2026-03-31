package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dylanbr0wn/coach/internal/types"
)

// DetectAgents finds which coding agents are installed on the current system.
func DetectAgents(overrideDir string) ([]types.DetectedAgent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return DetectAgentsInHome(home, overrideDir)
}

// DetectAgentsInHome finds agents using the given home directory.
// overrideDir is optional — pass "" to use embedded registry only.
func DetectAgentsInHome(home string, overrideDir ...string) ([]types.DetectedAgent, error) {
	override := ""
	if len(overrideDir) > 0 {
		override = overrideDir[0]
	}

	reg, err := LoadRegistry(override)
	if err != nil {
		return nil, err
	}

	var detected []types.DetectedAgent
	for key, agentCfg := range reg.Agents {
		resolvedDir := resolveHomePath(agentCfg.SkillDir, home)
		installed := dirExists(resolvedDir)

		detected = append(detected, types.DetectedAgent{
			Key:       key,
			Config:    agentCfg,
			Installed: installed,
			SkillDir:  resolvedDir,
		})
	}

	return detected, nil
}

// InstalledAgents returns only agents that are actually present on the system.
func InstalledAgents(agents []types.DetectedAgent) []types.DetectedAgent {
	var installed []types.DetectedAgent
	for _, a := range agents {
		if a.Installed {
			installed = append(installed, a)
		}
	}
	return installed
}

func resolveHomePath(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		resolved := filepath.Join(home, path[2:])
		// Preserve trailing slash if the original path had one.
		if strings.HasSuffix(path, "/") && !strings.HasSuffix(resolved, "/") {
			resolved += "/"
		}
		return resolved
	}
	return path
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
