// Package resolve provides skill resolution: finding skills by name by searching
// local project directories (walking up from WorkDir) and a global skills directory.
package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

// Scope controls where Resolve and List search for skills.
type Scope int

const (
	// ScopeAny searches local first, then global.
	ScopeAny Scope = iota
	// ScopeLocal searches only the local .coach/skills directory tree.
	ScopeLocal
	// ScopeGlobal searches only the global skills directory.
	ScopeGlobal
)

// Result describes a resolved skill.
type Result struct {
	Path  string // absolute path to SKILL.md
	Dir   string // absolute path to skill directory
	Name  string // skill name (directory name)
	Scope Scope  // where it was found (ScopeLocal or ScopeGlobal)
}

// Resolver finds and lists skills using a local project tree and a global directory.
type Resolver struct {
	// GlobalSkillsDir is the path to the global skills directory (e.g. ~/.coach/skills).
	GlobalSkillsDir string
	// WorkDir is the directory to start from when walking up for local skills.
	WorkDir string
}

// Resolve finds a skill by name. The search order depends on scope:
//   - ScopeAny: local first, then global.
//   - ScopeLocal: local only.
//   - ScopeGlobal: global only.
//
// Returns a descriptive error if the skill is not found.
func (r *Resolver) Resolve(name string, scope Scope) (Result, error) {
	if scope == ScopeLocal || scope == ScopeAny {
		if res, ok := r.findLocal(name); ok {
			return res, nil
		}
		if scope == ScopeLocal {
			return Result{}, fmt.Errorf("skill %q not found in local .coach/skills (searched from %s)", name, r.WorkDir)
		}
	}

	if scope == ScopeGlobal || scope == ScopeAny {
		if res, ok := r.findGlobal(name); ok {
			return res, nil
		}
		if scope == ScopeGlobal {
			return Result{}, fmt.Errorf("skill %q not found in global skills dir %s", name, r.GlobalSkillsDir)
		}
	}

	return Result{}, fmt.Errorf("skill %q not found (searched local .coach/skills and global %s)", name, r.GlobalSkillsDir)
}

// List returns all skills visible in the given scope. Local skills shadow global
// skills with the same name when using ScopeAny.
func (r *Resolver) List(scope Scope) ([]Result, error) {
	seen := make(map[string]bool)
	var results []Result

	if scope == ScopeLocal || scope == ScopeAny {
		localDir := r.localSkillsDir()
		if localDir != "" {
			local, err := listDir(localDir, ScopeLocal)
			if err != nil {
				return nil, fmt.Errorf("listing local skills: %w", err)
			}
			for _, res := range local {
				seen[res.Name] = true
				results = append(results, res)
			}
		}
	}

	if scope == ScopeGlobal || scope == ScopeAny {
		if dirExists(r.GlobalSkillsDir) {
			global, err := listDir(r.GlobalSkillsDir, ScopeGlobal)
			if err != nil {
				return nil, fmt.Errorf("listing global skills: %w", err)
			}
			for _, res := range global {
				if !seen[res.Name] {
					results = append(results, res)
				}
			}
		}
	}

	return results, nil
}

// TargetDir returns the directory where a new skill named name should be created.
// For ScopeLocal it returns <WorkDir>/.coach/skills/<name>.
// For ScopeGlobal it returns <GlobalSkillsDir>/<name>.
// For ScopeAny it defaults to the local target.
func (r *Resolver) TargetDir(name string, scope Scope) string {
	if scope == ScopeGlobal {
		return filepath.Join(r.GlobalSkillsDir, name)
	}
	return filepath.Join(r.WorkDir, ".coach", "skills", name)
}

// findLocal walks up from WorkDir looking for .coach/skills/<name>/SKILL.md.
func (r *Resolver) findLocal(name string) (Result, bool) {
	dir := r.WorkDir
	for {
		candidate := filepath.Join(dir, ".coach", "skills", name, "SKILL.md")
		if fileExists(candidate) {
			skillDir := filepath.Dir(candidate)
			return Result{
				Path:  candidate,
				Dir:   skillDir,
				Name:  name,
				Scope: ScopeLocal,
			}, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root.
			break
		}
		dir = parent
	}
	return Result{}, false
}

// findGlobal checks GlobalSkillsDir/<name>/SKILL.md.
func (r *Resolver) findGlobal(name string) (Result, bool) {
	candidate := filepath.Join(r.GlobalSkillsDir, name, "SKILL.md")
	if fileExists(candidate) {
		skillDir := filepath.Dir(candidate)
		return Result{
			Path:  candidate,
			Dir:   skillDir,
			Name:  name,
			Scope: ScopeGlobal,
		}, true
	}
	return Result{}, false
}

// localSkillsDir walks up from WorkDir and returns the first .coach/skills dir found,
// or empty string if none exists.
func (r *Resolver) localSkillsDir() string {
	dir := r.WorkDir
	for {
		candidate := filepath.Join(dir, ".coach", "skills")
		if dirExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// listDir enumerates immediate subdirectories of dir that contain a SKILL.md,
// assigning each the provided scope.
func listDir(dir string, scope Scope) ([]Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []Result
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
		if fileExists(skillPath) {
			skillDir := filepath.Join(dir, e.Name())
			results = append(results, Result{
				Path:  skillPath,
				Dir:   skillDir,
				Name:  e.Name(),
				Scope: scope,
			})
		}
	}
	return results, nil
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
