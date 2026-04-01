package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/types"
)

// Discover finds skill candidates from a source, or audits agent directories
// when installed is true.
//
// When installed is true, src is ignored and agents' skill directories are
// scanned for untracked or modified skills. When installed is false, src must
// be non-nil and is used to discover skills from a local path or remote repo.
func Discover(src *registry.Source, installed bool, agents []types.DetectedAgent, provenance *registry.InstalledSkills) ([]SkillCandidate, error) {
	if installed {
		return discoverInstalled(agents, provenance)
	}
	if src == nil {
		return nil, fmt.Errorf("source is required when not auditing installed skills")
	}
	return discoverSource(src)
}

// discoverSource discovers skills from a local path or remote repo.
func discoverSource(src *registry.Source) ([]SkillCandidate, error) {
	var localPath, sha string

	if src.Type == registry.SourceLocal {
		abs, err := filepath.Abs(src.Path)
		if err != nil {
			return nil, fmt.Errorf("resolving local path: %w", err)
		}
		localPath = abs
		sha = "local"
	} else {
		var err error
		localPath, sha, err = registry.FetchToCache(src)
		if err != nil {
			return nil, fmt.Errorf("fetching source: %w", err)
		}
	}

	skillPaths, err := registry.FindSkills(localPath)
	if err != nil {
		return nil, fmt.Errorf("finding skills: %w", err)
	}
	if len(skillPaths) == 0 {
		return nil, fmt.Errorf("no skills found in %s", src.Raw)
	}

	origin := OriginLocal
	if src.Type != registry.SourceLocal {
		origin = OriginRemote
	}

	candidates := make([]SkillCandidate, 0, len(skillPaths))
	for _, sp := range skillPaths {
		candidates = append(candidates, SkillCandidate{
			Path:   sp,
			Source: src.Raw,
			SHA:    sha,
			Origin: origin,
		})
	}
	return candidates, nil
}

// discoverInstalled audits agent skill directories for skills that are
// untracked (no provenance record) or modified (content differs from
// provenance hash).
func discoverInstalled(agents []types.DetectedAgent, provenance *registry.InstalledSkills) ([]SkillCandidate, error) {
	if provenance == nil {
		provenance = &registry.InstalledSkills{}
	}

	// Build lookup: skill name → content hash from provenance.
	hashByName := make(map[string]string)
	for _, s := range provenance.Skills {
		hashByName[s.Name] = s.ContentHash
	}

	// Track skills we've already seen to avoid duplicates across agents.
	seen := make(map[string]bool)
	var candidates []SkillCandidate

	for _, a := range agents {
		if !a.Installed || a.SkillDir == "" {
			continue
		}

		entries, err := os.ReadDir(a.SkillDir)
		if err != nil {
			continue // agent dir may not exist yet
		}

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			skillDir := filepath.Join(a.SkillDir, name)
			skillFile := filepath.Join(skillDir, "SKILL.md")

			if _, err := os.Stat(skillFile); err != nil {
				continue // not a skill directory
			}
			if seen[name] {
				continue
			}
			seen[name] = true

			hash, err := ContentHash(skillFile)
			if err != nil {
				continue // unreadable, skip
			}

			knownHash, tracked := hashByName[name]
			if tracked && knownHash != "" && knownHash == hash {
				continue // known and unchanged, skip
			}

			origin := OriginInstalledUntracked
			if tracked {
				origin = OriginInstalledModified
			}

			candidates = append(candidates, SkillCandidate{
				Path:   skillDir,
				Source: "installed:" + a.Config.Name,
				SHA:    "local",
				Origin: origin,
			})
		}
	}

	return candidates, nil
}

// ContentHash computes a hex-encoded SHA-256 hash of the file at path.
func ContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
