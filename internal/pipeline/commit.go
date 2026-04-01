package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/registry"
)

// CommitResult records the outcome of installing a single skill.
type CommitResult struct {
	Name   string
	Agents []string
	Err    error
}

// Commit installs selected skills to the scope directory, distributes them
// to configured agents, and records provenance.
func Commit(selected []VettedSkill, coachDir string, opts InstallOptions) ([]CommitResult, error) {
	scopeDir, err := resolveScopeDir(coachDir, opts.Scope)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(scopeDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating scope directory %s: %w", scopeDir, err)
	}

	regOpts := registry.InstallOptions{Copy: opts.Copy}
	var results []CommitResult

	for _, v := range selected {
		name := filepath.Base(v.Candidate.Path)
		if v.Skill != nil {
			name = v.Skill.Name
		}

		r := CommitResult{Name: name}

		// Install to scope directory.
		if err := registry.InstallSkill(v.Candidate.Path, scopeDir, regOpts); err != nil {
			r.Err = fmt.Errorf("installing to scope dir: %w", err)
			results = append(results, r)
			continue
		}

		// Compute content hash from the installed copy.
		installedSkillFile := filepath.Join(scopeDir, name, "SKILL.md")
		hash, hashErr := ContentHash(installedSkillFile)
		if hashErr != nil {
			// Non-fatal: install succeeded but we can't hash.
			hash = ""
		}

		// Distribute to each agent.
		installedDir := filepath.Join(scopeDir, name)
		for _, a := range opts.Agents {
			if !a.Installed || a.SkillDir == "" {
				continue
			}
			if err := registry.InstallSkill(installedDir, a.SkillDir, regOpts); err != nil {
				r.Err = fmt.Errorf("distributing to %s: %w", a.Config.Name, err)
				// Continue to other agents; record partial success.
				continue
			}
			r.Agents = append(r.Agents, a.Config.Name)
		}

		// Record provenance.
		score := 0
		if v.ScanResult != nil {
			score = v.ScanResult.Score
		}
		if err := registry.RecordInstall(
			coachDir, name, v.Candidate.Source, v.Candidate.SHA,
			hash, score, r.Agents,
		); err != nil {
			// Non-fatal: skill is installed, provenance just wasn't saved.
			if r.Err == nil {
				r.Err = fmt.Errorf("recording provenance: %w", err)
			}
		}

		results = append(results, r)
	}

	return results, nil
}

// resolveScopeDir returns the absolute path to the skill storage directory
// for the given scope.
func resolveScopeDir(coachDir, scope string) (string, error) {
	switch scope {
	case "global":
		return filepath.Join(coachDir, "skills"), nil
	case "local":
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
		return filepath.Join(cwd, ".coach", "skills"), nil
	default:
		// Fall back to config default.
		cfg, err := config.Load(coachDir)
		if err != nil {
			return filepath.Join(coachDir, "skills"), nil
		}
		if cfg.DefaultScope == "local" {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("getting working directory: %w", err)
			}
			return filepath.Join(cwd, ".coach", "skills"), nil
		}
		return filepath.Join(coachDir, "skills"), nil
	}
}
