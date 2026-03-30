// Package distribute handles symlinking skill directories into agent skill directories.
package distribute

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/pkg"
)

// Status describes the outcome of a distribution operation for one agent.
type Status int

const (
	StatusCreated  Status = iota // Symlink was newly created.
	StatusUpdated                // Stale symlink was replaced.
	StatusUpToDate               // Symlink already points to the correct target.
	StatusSkipped                // Agent not installed; nothing to do.
)

// String returns a human-readable label for the status.
func (s Status) String() string {
	switch s {
	case StatusCreated:
		return "created"
	case StatusUpdated:
		return "updated"
	case StatusUpToDate:
		return "up-to-date"
	case StatusSkipped:
		return "skipped"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// DistResult records the outcome of distributing a skill to one agent.
type DistResult struct {
	Agent  string
	Path   string
	Status Status
}

// Distribute symlinks skillDir into each installed agent's skill directory as
// <agent.SkillDir>/<skillName>. Agents that are not installed are skipped.
func Distribute(skillDir string, skillName string, agents []pkg.DetectedAgent) ([]DistResult, error) {
	results := make([]DistResult, 0, len(agents))

	for _, agent := range agents {
		if !agent.Installed {
			results = append(results, DistResult{
				Agent:  agent.Config.Name,
				Path:   "",
				Status: StatusSkipped,
			})
			continue
		}

		linkPath := filepath.Join(agent.SkillDir, skillName)

		// Ensure the parent directory exists.
		if err := os.MkdirAll(agent.SkillDir, 0o755); err != nil {
			return nil, fmt.Errorf("create skill dir for %s: %w", agent.Config.Name, err)
		}

		existing, err := os.Readlink(linkPath)
		if err == nil {
			// Symlink exists.
			if existing == skillDir {
				results = append(results, DistResult{
					Agent:  agent.Config.Name,
					Path:   linkPath,
					Status: StatusUpToDate,
				})
				continue
			}
			// Points elsewhere — remove and recreate.
			if removeErr := os.Remove(linkPath); removeErr != nil {
				return nil, fmt.Errorf("remove stale symlink for %s: %w", agent.Config.Name, removeErr)
			}
			if symlinkErr := os.Symlink(skillDir, linkPath); symlinkErr != nil {
				return nil, fmt.Errorf("create symlink for %s: %w", agent.Config.Name, symlinkErr)
			}
			results = append(results, DistResult{
				Agent:  agent.Config.Name,
				Path:   linkPath,
				Status: StatusUpdated,
			})
			continue
		}

		// No symlink yet (or some other error reading it — treat as absent).
		if symlinkErr := os.Symlink(skillDir, linkPath); symlinkErr != nil {
			return nil, fmt.Errorf("create symlink for %s: %w", agent.Config.Name, symlinkErr)
		}
		results = append(results, DistResult{
			Agent:  agent.Config.Name,
			Path:   linkPath,
			Status: StatusCreated,
		})
	}

	return results, nil
}

// FilterAgentsByNames returns only those agents whose Key or Config.Name appears in names.
// This allows matching by registry key (e.g., "claude-code") or display name (e.g., "Claude Code").
func FilterAgentsByNames(agents []pkg.DetectedAgent, names []string) []pkg.DetectedAgent {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}

	filtered := make([]pkg.DetectedAgent, 0, len(agents))
	for _, a := range agents {
		if _, ok := set[a.Config.Name]; ok {
			filtered = append(filtered, a)
			continue
		}
		if _, ok := set[a.Key]; ok {
			filtered = append(filtered, a)
		}
	}
	return filtered
}
