package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dylan/coach/pkg"
)

// DetectAgents finds which coding agents are installed on the current system.
func DetectAgents(overrideDir string) ([]pkg.DetectedAgent, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return DetectAgentsInHome(home, overrideDir)
}

// DetectAgentsInHome finds agents using the given home directory.
// overrideDir is optional — pass "" to use embedded registry only.
func DetectAgentsInHome(home string, overrideDir ...string) ([]pkg.DetectedAgent, error) {
	override := ""
	if len(overrideDir) > 0 {
		override = overrideDir[0]
	}

	reg, err := LoadRegistry(override)
	if err != nil {
		return nil, err
	}

	var detected []pkg.DetectedAgent
	for _, agentCfg := range reg.Agents {
		resolvedDir := resolveHomePath(agentCfg.SkillDir, home)
		installed := dirExists(resolvedDir)

		detected = append(detected, pkg.DetectedAgent{
			Config:    agentCfg,
			Installed: installed,
			SkillDir:  resolvedDir,
		})
	}

	return detected, nil
}

// InstalledAgents returns only agents that are actually present on the system.
func InstalledAgents(agents []pkg.DetectedAgent) []pkg.DetectedAgent {
	var installed []pkg.DetectedAgent
	for _, a := range agents {
		if a.Installed {
			installed = append(installed, a)
		}
	}
	return installed
}

func resolveHomePath(path string, home string) string {
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
