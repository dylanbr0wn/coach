# Skill Editing & Authoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add skill editing, LLM-assisted authoring, distribution, and configuration commands to coach CLI.

**Architecture:** Extend the existing Cobra CLI with four new commands (`edit`, `generate`, `config`, `sync`) backed by three new internal packages (`resolve`, `distribute`, `llm`). Skills are stored in coach-managed directories (local `.coach/skills/` or global `~/.coach/skills/`) and symlinked to agent directories for distribution.

**Tech Stack:** Go 1.24, Cobra CLI, charmbracelet/huh (forms), charmbracelet/lipgloss (styling), gopkg.in/yaml.v3

**Spec:** `docs/superpowers/specs/2026-03-29-skill-editing-design.md`

---

### Task 1: Extend Config with New Fields

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for new config fields**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	cfg := Config{
		RulesSource:    "https://github.com/coach-dev/security-rules",
		TrustedSources: []string{"github.com/org"},
		DefaultAgents:  []string{"claude"},
		DistributeTo:   []string{"claude", "cursor"},
		LLMCli:         "claude",
		DefaultScope:   "global",
	}

	err := SaveTo(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.LLMCli != "claude" {
		t.Errorf("LLMCli = %q, want %q", loaded.LLMCli, "claude")
	}
	if loaded.DefaultScope != "global" {
		t.Errorf("DefaultScope = %q, want %q", loaded.DefaultScope, "global")
	}
	if len(loaded.DistributeTo) != 2 {
		t.Errorf("DistributeTo len = %d, want 2", len(loaded.DistributeTo))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.LLMCli != "claude" {
		t.Errorf("default LLMCli = %q, want %q", cfg.LLMCli, "claude")
	}
	if cfg.DefaultScope != "global" {
		t.Errorf("default DefaultScope = %q, want %q", cfg.DefaultScope, "global")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dylan/vault/coach && go test ./internal/config/ -v -run TestConfig`
Expected: FAIL — `SaveTo`, `LoadFrom`, new fields not defined

- [ ] **Step 3: Add new fields and functions to config**

Add three new fields to the `Config` struct in `internal/config/config.go`:

```go
type Config struct {
	RulesSource    string   `yaml:"rules_source"`
	TrustedSources []string `yaml:"trusted_sources,omitempty"`
	DefaultAgents  []string `yaml:"default_agents,omitempty"`
	DistributeTo   []string `yaml:"distribute_to,omitempty"`
	LLMCli         string   `yaml:"llm_cli,omitempty"`
	DefaultScope   string   `yaml:"default_scope,omitempty"`
}
```

Update `defaultConfig` to set defaults for the new fields:

```go
func defaultConfig() Config {
	return Config{
		RulesSource:  "https://github.com/coach-dev/security-rules",
		LLMCli:       "claude",
		DefaultScope: "global",
	}
}
```

Add `SaveTo` and `LoadFrom` functions that accept explicit paths (the existing `Save`/`Load` can call these with the default path):

```go
func SaveTo(cfg Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadFrom(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}
```

Update existing `Save` and `Load` to delegate:

```go
func Save(cfg Config) error {
	dir, err := DefaultCoachDir()
	if err != nil {
		return err
	}
	return SaveTo(cfg, filepath.Join(dir, "config.yaml"))
}

func Load() (Config, error) {
	dir, err := DefaultCoachDir()
	if err != nil {
		return Config{}, err
	}
	return LoadFrom(filepath.Join(dir, "config.yaml"))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/vault/coach && go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add distribute_to, llm_cli, default_scope fields"
```

---

### Task 2: Skill Resolution Package

**Files:**
- Create: `internal/resolve/resolve.go`
- Create: `internal/resolve/resolve_test.go`

- [ ] **Step 1: Write failing tests for skill resolution**

```go
// internal/resolve/resolve_test.go
package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, dir, name string) string {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: test skill\n---\nBody here.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return skillDir
}

func TestResolveLocal(t *testing.T) {
	globalDir := filepath.Join(t.TempDir(), "global", "skills")
	localRoot := t.TempDir()
	localSkillsDir := filepath.Join(localRoot, ".coach", "skills")
	writeSkill(t, localSkillsDir, "my-skill")

	r := Resolver{GlobalSkillsDir: globalDir, WorkDir: localRoot}
	result, err := r.Resolve("my-skill", ScopeAny)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if result.Scope != ScopeLocal {
		t.Errorf("scope = %v, want ScopeLocal", result.Scope)
	}
	if filepath.Base(filepath.Dir(result.Path)) != "my-skill" {
		t.Errorf("path = %q, want dir named my-skill", result.Path)
	}
}

func TestResolveGlobal(t *testing.T) {
	globalDir := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, globalDir, "global-skill")
	localRoot := t.TempDir() // no .coach/skills here

	r := Resolver{GlobalSkillsDir: globalDir, WorkDir: localRoot}
	result, err := r.Resolve("global-skill", ScopeAny)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if result.Scope != ScopeGlobal {
		t.Errorf("scope = %v, want ScopeGlobal", result.Scope)
	}
}

func TestResolveLocalOverridesGlobal(t *testing.T) {
	globalDir := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, globalDir, "shared-skill")

	localRoot := t.TempDir()
	localSkillsDir := filepath.Join(localRoot, ".coach", "skills")
	writeSkill(t, localSkillsDir, "shared-skill")

	r := Resolver{GlobalSkillsDir: globalDir, WorkDir: localRoot}
	result, err := r.Resolve("shared-skill", ScopeAny)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if result.Scope != ScopeLocal {
		t.Errorf("scope = %v, want ScopeLocal (local should override global)", result.Scope)
	}
}

func TestResolveNotFound(t *testing.T) {
	globalDir := filepath.Join(t.TempDir(), "skills")
	localRoot := t.TempDir()

	r := Resolver{GlobalSkillsDir: globalDir, WorkDir: localRoot}
	_, err := r.Resolve("nonexistent", ScopeAny)
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestResolveForcedScope(t *testing.T) {
	globalDir := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, globalDir, "only-global")

	localRoot := t.TempDir()

	r := Resolver{GlobalSkillsDir: globalDir, WorkDir: localRoot}
	_, err := r.Resolve("only-global", ScopeLocal)
	if err == nil {
		t.Fatal("expected error when forcing local scope but skill only exists globally")
	}

	result, err := r.Resolve("only-global", ScopeGlobal)
	if err != nil {
		t.Fatalf("Resolve with ScopeGlobal failed: %v", err)
	}
	if result.Scope != ScopeGlobal {
		t.Errorf("scope = %v, want ScopeGlobal", result.Scope)
	}
}

func TestListSkills(t *testing.T) {
	globalDir := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, globalDir, "skill-a")
	writeSkill(t, globalDir, "skill-b")

	localRoot := t.TempDir()
	localSkillsDir := filepath.Join(localRoot, ".coach", "skills")
	writeSkill(t, localSkillsDir, "skill-c")

	r := Resolver{GlobalSkillsDir: globalDir, WorkDir: localRoot}
	skills, err := r.List(ScopeAny)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("len = %d, want 3", len(skills))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/dylan/vault/coach && go test ./internal/resolve/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement the resolve package**

```go
// internal/resolve/resolve.go
package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

type Scope int

const (
	ScopeAny    Scope = iota
	ScopeLocal
	ScopeGlobal
)

type Result struct {
	Path  string // absolute path to SKILL.md
	Dir   string // absolute path to skill directory
	Name  string // skill name (directory name)
	Scope Scope  // where it was found
}

type Resolver struct {
	GlobalSkillsDir string // e.g. ~/.coach/skills
	WorkDir         string // current working directory
}

func (r *Resolver) Resolve(name string, scope Scope) (Result, error) {
	if scope == ScopeAny || scope == ScopeLocal {
		if result, ok := r.findLocal(name); ok {
			return result, nil
		}
		if scope == ScopeLocal {
			return Result{}, fmt.Errorf("skill %q not found in local .coach/skills/", name)
		}
	}

	if scope == ScopeAny || scope == ScopeGlobal {
		if result, ok := r.findGlobal(name); ok {
			return result, nil
		}
		if scope == ScopeGlobal {
			return Result{}, fmt.Errorf("skill %q not found in global skills", name)
		}
	}

	return Result{}, fmt.Errorf("skill %q not found. Did you mean 'coach init skill' or 'coach generate %s'?", name, name)
}

func (r *Resolver) findLocal(name string) (Result, bool) {
	dir := r.WorkDir
	for {
		candidate := filepath.Join(dir, ".coach", "skills", name, "SKILL.md")
		if fileExists(candidate) {
			skillDir := filepath.Dir(candidate)
			return Result{Path: candidate, Dir: skillDir, Name: name, Scope: ScopeLocal}, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return Result{}, false
}

func (r *Resolver) findGlobal(name string) (Result, bool) {
	candidate := filepath.Join(r.GlobalSkillsDir, name, "SKILL.md")
	if fileExists(candidate) {
		skillDir := filepath.Dir(candidate)
		return Result{Path: candidate, Dir: skillDir, Name: name, Scope: ScopeGlobal}, true
	}
	return Result{}, false
}

// List returns all managed skills visible from the current context.
func (r *Resolver) List(scope Scope) ([]Result, error) {
	var results []Result

	if scope == ScopeAny || scope == ScopeLocal {
		locals, err := r.listDir(r.localSkillsDir(), ScopeLocal)
		if err != nil {
			return nil, err
		}
		results = append(results, locals...)
	}

	if scope == ScopeAny || scope == ScopeGlobal {
		globals, err := r.listDir(r.GlobalSkillsDir, ScopeGlobal)
		if err != nil {
			return nil, err
		}
		// Don't add globals that are shadowed by locals
		seen := make(map[string]bool)
		for _, r := range results {
			seen[r.Name] = true
		}
		for _, g := range globals {
			if !seen[g.Name] {
				results = append(results, g)
			}
		}
	}

	return results, nil
}

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
	return filepath.Join(r.WorkDir, ".coach", "skills")
}

// TargetDir returns the directory where a new skill should be created.
func (r *Resolver) TargetDir(name string, scope Scope) string {
	if scope == ScopeLocal {
		return filepath.Join(r.localSkillsDir(), name)
	}
	return filepath.Join(r.GlobalSkillsDir, name)
}

func (r *Resolver) listDir(dir string, scope Scope) ([]Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var results []Result
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
		if fileExists(skillFile) {
			results = append(results, Result{
				Path:  skillFile,
				Dir:   filepath.Join(dir, e.Name()),
				Name:  e.Name(),
				Scope: scope,
			})
		}
	}
	return results, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/vault/coach && go test ./internal/resolve/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/resolve/
git commit -m "feat(resolve): add skill resolution with local/global scope"
```

---

### Task 3: Distribution Package

**Files:**
- Create: `internal/distribute/distribute.go`
- Create: `internal/distribute/distribute_test.go`

- [ ] **Step 1: Write failing tests for distribution**

```go
// internal/distribute/distribute_test.go
package distribute

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/pkg"
)

func TestDistributeCreatesSymlink(t *testing.T) {
	skillDir := filepath.Join(t.TempDir(), "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentSkillDir := filepath.Join(t.TempDir(), "agent-skills")
	if err := os.MkdirAll(agentSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents := []pkg.DetectedAgent{
		{
			Config:    pkg.AgentConfig{Name: "test-agent", SkillDir: agentSkillDir + "/"},
			Installed: true,
			SkillDir:  agentSkillDir,
		},
	}

	results, err := Distribute(skillDir, "my-skill", agents)
	if err != nil {
		t.Fatalf("Distribute failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Status != StatusCreated {
		t.Errorf("status = %v, want StatusCreated", results[0].Status)
	}

	linkPath := filepath.Join(agentSkillDir, "my-skill")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if target != skillDir {
		t.Errorf("symlink target = %q, want %q", target, skillDir)
	}
}

func TestDistributeAlreadyLinked(t *testing.T) {
	skillDir := filepath.Join(t.TempDir(), "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agentSkillDir := filepath.Join(t.TempDir(), "agent-skills")
	if err := os.MkdirAll(agentSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create the symlink
	if err := os.Symlink(skillDir, filepath.Join(agentSkillDir, "my-skill")); err != nil {
		t.Fatal(err)
	}

	agents := []pkg.DetectedAgent{
		{
			Config:    pkg.AgentConfig{Name: "test-agent", SkillDir: agentSkillDir + "/"},
			Installed: true,
			SkillDir:  agentSkillDir,
		},
	}

	results, err := Distribute(skillDir, "my-skill", agents)
	if err != nil {
		t.Fatalf("Distribute failed: %v", err)
	}
	if results[0].Status != StatusUpToDate {
		t.Errorf("status = %v, want StatusUpToDate", results[0].Status)
	}
}

func TestDistributeSkipsUninstalledAgents(t *testing.T) {
	skillDir := filepath.Join(t.TempDir(), "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents := []pkg.DetectedAgent{
		{
			Config:    pkg.AgentConfig{Name: "missing-agent", SkillDir: "/nonexistent/"},
			Installed: false,
			SkillDir:  "/nonexistent",
		},
	}

	results, err := Distribute(skillDir, "my-skill", agents)
	if err != nil {
		t.Fatalf("Distribute failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Status != StatusSkipped {
		t.Errorf("status = %v, want StatusSkipped", results[0].Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/dylan/vault/coach && go test ./internal/distribute/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement the distribute package**

```go
// internal/distribute/distribute.go
package distribute

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/pkg"
)

type Status int

const (
	StatusCreated  Status = iota
	StatusUpdated
	StatusUpToDate
	StatusSkipped
)

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
		return "unknown"
	}
}

type DistResult struct {
	Agent  string
	Path   string
	Status Status
}

// Distribute symlinks a skill directory into each agent's skill directory.
func Distribute(skillDir string, skillName string, agents []pkg.DetectedAgent) ([]DistResult, error) {
	var results []DistResult

	for _, a := range agents {
		if !a.Installed {
			results = append(results, DistResult{
				Agent:  a.Config.Name,
				Status: StatusSkipped,
			})
			continue
		}

		linkPath := filepath.Join(a.SkillDir, skillName)

		// Check if symlink already exists and points to the right place
		existing, err := os.Readlink(linkPath)
		if err == nil {
			if existing == skillDir {
				results = append(results, DistResult{
					Agent:  a.Config.Name,
					Path:   linkPath,
					Status: StatusUpToDate,
				})
				continue
			}
			// Points to wrong place — remove and recreate
			if err := os.Remove(linkPath); err != nil {
				return nil, fmt.Errorf("removing stale symlink %s: %w", linkPath, err)
			}
		}

		// Remove if it's a regular file/dir (not a symlink)
		if _, err := os.Lstat(linkPath); err == nil {
			if err := os.RemoveAll(linkPath); err != nil {
				return nil, fmt.Errorf("removing existing %s: %w", linkPath, err)
			}
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return nil, fmt.Errorf("creating agent skill dir: %w", err)
		}

		if err := os.Symlink(skillDir, linkPath); err != nil {
			return nil, fmt.Errorf("creating symlink %s -> %s: %w", linkPath, skillDir, err)
		}

		status := StatusCreated
		if existing != "" {
			status = StatusUpdated
		}

		results = append(results, DistResult{
			Agent:  a.Config.Name,
			Path:   linkPath,
			Status: status,
		})
	}

	return results, nil
}

// FilterAgentsByNames returns only agents whose names appear in the given list.
func FilterAgentsByNames(agents []pkg.DetectedAgent, names []string) []pkg.DetectedAgent {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	var filtered []pkg.DetectedAgent
	for _, a := range agents {
		if nameSet[a.Config.Name] {
			filtered = append(filtered, a)
		}
	}
	return filtered
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/vault/coach && go test ./internal/distribute/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/distribute/
git commit -m "feat(distribute): add symlink-based skill distribution to agents"
```

---

### Task 4: `coach config` Command

**Files:**
- Create: `cmd/config.go`
- Create: `cmd/config_test.go`

- [ ] **Step 1: Write failing test for config command**

```go
// cmd/config_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/config"
)

func TestConfigSetAndGet(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Test setting llm_cli
	err := setConfigValue(configPath, "llm-cli", "codex")
	if err != nil {
		t.Fatalf("setConfigValue failed: %v", err)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if cfg.LLMCli != "codex" {
		t.Errorf("LLMCli = %q, want %q", cfg.LLMCli, "codex")
	}

	// Test setting distribute-to
	err = setConfigValue(configPath, "distribute-to", "claude,cursor")
	if err != nil {
		t.Fatalf("setConfigValue failed: %v", err)
	}

	cfg, err = config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if len(cfg.DistributeTo) != 2 || cfg.DistributeTo[0] != "claude" {
		t.Errorf("DistributeTo = %v, want [claude cursor]", cfg.DistributeTo)
	}
}

func TestConfigSetInvalidKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	err := setConfigValue(configPath, "invalid-key", "value")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestConfigGetValue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	cfg := config.Config{LLMCli: "claude", DistributeTo: []string{"claude", "cursor"}}
	if err := config.SaveTo(cfg, configPath); err != nil {
		t.Fatal(err)
	}

	val, err := getConfigValue(configPath, "llm-cli")
	if err != nil {
		t.Fatalf("getConfigValue failed: %v", err)
	}
	if val != "claude" {
		t.Errorf("got %q, want %q", val, "claude")
	}

	val, err = getConfigValue(configPath, "distribute-to")
	if err != nil {
		t.Fatalf("getConfigValue failed: %v", err)
	}
	if val != "claude,cursor" {
		t.Errorf("got %q, want %q", val, "claude,cursor")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dylan/vault/coach && go test ./cmd/ -v -run TestConfig`
Expected: FAIL — `setConfigValue`, `getConfigValue` not defined

- [ ] **Step 3: Implement config command**

```go
// cmd/config.go
package cmd

import (
	"fmt"
	"strings"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage coach configuration",
	Long: `View and modify coach configuration.

Examples:
  coach config set distribute-to claude,cursor    # Set distribution targets
  coach config set llm-cli claude                  # Set default LLM CLI
  coach config get distribute-to                   # Show current value
  coach config get llm-cli                         # Show LLM CLI setting`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		coachDir, err := config.DefaultCoachDir()
		if err != nil {
			return err
		}
		configPath := coachDir + "/config.yaml"
		if err := setConfigValue(configPath, args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Set %s = %s\n", args[0], args[1])
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		coachDir, err := config.DefaultCoachDir()
		if err != nil {
			return err
		}
		configPath := coachDir + "/config.yaml"
		val, err := getConfigValue(configPath, args[0])
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	},
}

var validConfigKeys = map[string]bool{
	"distribute-to": true,
	"llm-cli":       true,
	"default-scope": true,
}

func setConfigValue(configPath, key, value string) error {
	if !validConfigKeys[key] {
		return fmt.Errorf("unknown config key %q (valid keys: distribute-to, llm-cli, default-scope)", key)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		return err
	}

	switch key {
	case "distribute-to":
		cfg.DistributeTo = strings.Split(value, ",")
		for i := range cfg.DistributeTo {
			cfg.DistributeTo[i] = strings.TrimSpace(cfg.DistributeTo[i])
		}
	case "llm-cli":
		cfg.LLMCli = value
	case "default-scope":
		if value != "global" && value != "local" {
			return fmt.Errorf("default-scope must be 'global' or 'local', got %q", value)
		}
		cfg.DefaultScope = value
	}

	return config.SaveTo(cfg, configPath)
}

func getConfigValue(configPath, key string) (string, error) {
	if !validConfigKeys[key] {
		return "", fmt.Errorf("unknown config key %q (valid keys: distribute-to, llm-cli, default-scope)", key)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		return "", err
	}

	switch key {
	case "distribute-to":
		return strings.Join(cfg.DistributeTo, ","), nil
	case "llm-cli":
		return cfg.LLMCli, nil
	case "default-scope":
		return cfg.DefaultScope, nil
	}

	return "", fmt.Errorf("unknown key %q", key)
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	rootCmd.AddCommand(configCmd)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/vault/coach && go test ./cmd/ -v -run TestConfig`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/config.go cmd/config_test.go
git commit -m "feat: add coach config set/get command"
```

---

### Task 5: `coach edit` Command

**Files:**
- Create: `cmd/edit.go`
- Create: `cmd/edit_test.go`

- [ ] **Step 1: Write failing tests for edit logic**

```go
// cmd/edit_test.go
package cmd

import (
	"os"
	"testing"
)

func TestGetEditor(t *testing.T) {
	// Save and restore EDITOR
	orig := os.Getenv("EDITOR")
	defer os.Setenv("EDITOR", orig)

	os.Setenv("EDITOR", "nvim")
	editor, err := getEditor()
	if err != nil {
		t.Fatalf("getEditor failed: %v", err)
	}
	if editor != "nvim" {
		t.Errorf("editor = %q, want %q", editor, "nvim")
	}

	os.Setenv("EDITOR", "")
	os.Setenv("VISUAL", "code")
	editor, err = getEditor()
	if err != nil {
		t.Fatalf("getEditor failed: %v", err)
	}
	if editor != "code" {
		t.Errorf("editor = %q, want %q", editor, "code")
	}
	os.Setenv("VISUAL", "")
}

func TestDetectChanges(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "skill-*.md")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("---\nname: test\n---\nbody")
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	hash, err := fileHash(f.Name())
	if err != nil {
		t.Fatalf("fileHash failed: %v", err)
	}

	changed, err := fileChanged(f.Name(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("file should not be changed")
	}

	// Modify file
	os.WriteFile(f.Name(), []byte("---\nname: test\n---\nupdated body"), 0o644)
	changed, err = fileChanged(f.Name(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("file should be changed")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/dylan/vault/coach && go test ./cmd/ -v -run "TestGetEditor|TestDetectChanges"`
Expected: FAIL — `getEditor`, `fileHash`, `fileChanged` not defined

- [ ] **Step 3: Implement edit command**

```go
// cmd/edit.go
package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var editGlobal bool
var editLocal bool

var editCmd = &cobra.Command{
	Use:   "edit <skill-name>",
	Short: "Open a skill in your editor",
	Long: `Open a skill's SKILL.md in $EDITOR and validate on save.

After closing the editor, coach runs lint to check for issues.
If problems are found, you'll be prompted to re-open and fix them.

Examples:
  coach edit code-reviewer          # Open in $EDITOR, lint on save
  coach edit code-reviewer -g       # Edit the global version
  coach edit deploy-check -l        # Edit the local (project) version`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func runEdit(cmd *cobra.Command, args []string) error {
	skillName := args[0]

	scope := resolve.ScopeAny
	if editGlobal {
		scope = resolve.ScopeGlobal
	} else if editLocal {
		scope = resolve.ScopeLocal
	}

	coachDir, err := config.DefaultCoachDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         cwd,
	}

	result, err := r.Resolve(skillName, scope)
	if err != nil {
		return err
	}

	editor, err := getEditor()
	if err != nil {
		return err
	}

	// Hash before editing
	beforeHash, err := fileHash(result.Path)
	if err != nil {
		return fmt.Errorf("reading skill file: %w", err)
	}

	for {
		// Open editor
		editorCmd := exec.Command(editor, result.Path)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Check for changes
		changed, err := fileChanged(result.Path, beforeHash)
		if err != nil {
			return err
		}
		if !changed {
			fmt.Println("No changes detected.")
			return nil
		}

		// Lint
		s, err := skill.Parse(result.Dir)
		if err != nil {
			fmt.Printf("\n%s\n", ui.ErrorStyle.Render("Parse error: "+err.Error()))
			if !promptReopen() {
				return nil
			}
			continue
		}

		errors := skill.Validate(s)
		if len(errors) > 0 {
			fmt.Printf("\n%s\n", ui.ErrorStyle.Render("Validation issues:"))
			for _, e := range errors {
				fmt.Printf("  • %s\n", e)
			}
			if !promptReopen() {
				return nil
			}
			continue
		}

		fmt.Printf("\n%s Skill %q updated and passes validation.\n", ui.SuccessStyle.Render("✓"), skillName)
		return nil
	}
}

func getEditor() (string, error) {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor, nil
	}
	// Check if vi is available
	if _, err := exec.LookPath("vi"); err == nil {
		return "vi", nil
	}
	return "", fmt.Errorf("no editor found: set $EDITOR or $VISUAL environment variable")
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileChanged(path, previousHash string) (bool, error) {
	current, err := fileHash(path)
	if err != nil {
		return false, err
	}
	return current != previousHash, nil
}

func promptReopen() bool {
	fmt.Print("Re-open to fix? [Y/n] ")
	var response string
	fmt.Scanln(&response)
	return response == "" || response == "y" || response == "Y"
}

func init() {
	editCmd.Flags().BoolVarP(&editGlobal, "global", "g", false, "Edit from global skills")
	editCmd.Flags().BoolVarP(&editLocal, "local", "l", false, "Edit from local project skills")
	rootCmd.AddCommand(editCmd)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/vault/coach && go test ./cmd/ -v -run "TestGetEditor|TestDetectChanges"`
Expected: PASS

- [ ] **Step 5: Manual smoke test**

Run: `cd /Users/dylan/vault/coach && go run . edit skill-coach`
Expected: Should error with resolution message (skill-coach isn't in `~/.coach/skills/` yet). This confirms the command is wired up and resolution works.

- [ ] **Step 6: Commit**

```bash
git add cmd/edit.go cmd/edit_test.go
git commit -m "feat: add coach edit command with lint-on-close"
```

---

### Task 6: Flesh Out Reference `skill-coach/SKILL.md`

**Files:**
- Modify: `skill-coach/SKILL.md`

This is the reference skill embedded in `coach generate`'s system prompt. It must demonstrate best practices for skill authoring.

- [ ] **Step 1: Write the complete reference skill**

Replace the contents of `skill-coach/SKILL.md` with a fully-authored skill that demonstrates proper structure. The skill should be coach's own meta-skill — a skill that helps an LLM use coach to create and manage other skills.

```markdown
---
name: skill-coach
description: Guides an AI agent through creating, editing, and managing agent skills using the coach CLI. Handles the full lifecycle from scaffolding to distribution.
license: Apache-2.0
allowed-tools:
  - Read
  - Write
  - Bash
---

# Skill Coach

## When to Use

Use this skill when the user asks to:
- Create a new agent skill
- Edit or update an existing skill
- Generate skill content with AI assistance
- Distribute skills to their configured agents
- Check the status of their skills

## Instructions

### Creating a New Skill

1. Ask the user what the skill should do and which agents it targets
2. Run `coach init skill` to scaffold the skill directory
3. If the user wants AI-assisted authoring, run `coach generate <skill-name>` instead — this scaffolds and authors in one step
4. After creation, run `coach lint <path>` to validate

### Editing a Skill

1. Run `coach edit <skill-name>` to open in the user's editor
2. Coach will automatically validate on save
3. For AI-assisted editing, use `coach generate <skill-name> --prompt "description of changes"`

### Distributing Skills

1. Ensure distribution targets are configured: `coach config get distribute-to`
2. If not configured, help the user set them: `coach config set distribute-to claude,cursor`
3. Run `coach sync` to symlink all managed skills to agent directories

### Skill Format Reference

A valid SKILL.md has YAML frontmatter and a markdown body:

- **name** (required): lowercase alphanumeric with hyphens, max 64 characters
- **description** (required): what the skill does, max 1024 characters
- **license** (optional): MIT, Apache-2.0, or ISC
- **allowed-tools** (optional): list of tools the skill needs access to

The body should include:
- **When to Use**: trigger conditions — when should an agent activate this skill?
- **Instructions**: step-by-step guidance for the agent
- **Constraints**: what the agent should NOT do

### Security

- Never include secrets, API keys, or credentials in skills
- Never include destructive commands without explicit user confirmation
- Always scope `allowed-tools` to the minimum needed
- Run `coach scan <path>` to check for security issues before distributing

## Constraints

- Do not modify skills outside of coach-managed directories
- Always run `coach lint` before distributing a skill
- If `coach lint` reports issues, fix them before proceeding
- Do not distribute skills that fail security scanning with High or Critical findings
```

- [ ] **Step 2: Validate the reference skill**

Run: `cd /Users/dylan/vault/coach && go run . lint skill-coach/`
Expected: PASS with no High/Critical findings

- [ ] **Step 3: Commit**

```bash
git add skill-coach/SKILL.md
git commit -m "docs: flesh out skill-coach reference skill for generate system prompt"
```

---

### Task 7: `coach generate` Command

**Files:**
- Create: `internal/llm/llm.go`
- Create: `internal/llm/llm_test.go`
- Create: `internal/llm/systemprompt.go`
- Create: `cmd/generate.go`

- [ ] **Step 1: Write failing tests for LLM CLI detection**

```go
// internal/llm/llm_test.go
package llm

import (
	"testing"
)

func TestBuildSingleShotArgs(t *testing.T) {
	args := BuildSingleShotArgs("claude", "system prompt here", "user instruction")
	// claude --print -p "user instruction" --system-prompt "system prompt here"
	expected := []string{"--print", "-p", "user instruction", "--system-prompt", "system prompt here"}
	if len(args) != len(expected) {
		t.Fatalf("args len = %d, want %d: %v", len(args), len(expected), args)
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("args[%d] = %q, want %q", i, arg, expected[i])
		}
	}
}

func TestBuildInteractiveArgs(t *testing.T) {
	args := BuildInteractiveArgs("claude", "system prompt here")
	expected := []string{"--system-prompt", "system prompt here"}
	if len(args) != len(expected) {
		t.Fatalf("args len = %d, want %d: %v", len(args), len(expected), args)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt("", "")
	if prompt == "" {
		t.Fatal("system prompt should not be empty")
	}
	// Should contain the format spec
	if !containsString(prompt, "SKILL.md") {
		t.Error("system prompt should mention SKILL.md")
	}
}

func TestBuildSystemPromptWithExisting(t *testing.T) {
	existing := "---\nname: test\ndescription: test skill\n---\nOld body."
	prompt := BuildSystemPrompt(existing, "")
	if !containsString(prompt, "Old body") {
		t.Error("system prompt should include existing skill content")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/dylan/vault/coach && go test ./internal/llm/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement LLM package — system prompt**

```go
// internal/llm/systemprompt.go
package llm

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed reference_skill.md
var referenceSkill string

const formatSpec = `## SKILL.md Format

A skill is a directory containing a SKILL.md file with YAML frontmatter and a markdown body.

### Frontmatter (YAML between --- delimiters)

Required fields:
- name: lowercase alphanumeric with hyphens only, max 64 characters
- description: what the skill does, max 1024 characters

Optional fields:
- license: MIT, Apache-2.0, or ISC
- allowed-tools: list of tool names the skill needs (e.g., Read, Write, Bash, Grep, Glob, Edit)

### Body (markdown after frontmatter)

The body contains the actual skill instructions. A good skill body includes:
- **When to Use**: Clear trigger conditions — when should an agent activate this skill?
- **Instructions**: Step-by-step guidance organized by workflow or task type
- **Constraints**: What the agent should NOT do

### Security Rules
- Never include secrets, API keys, or credentials
- Never include destructive commands without requiring user confirmation
- Scope allowed-tools to the minimum needed`

func BuildSystemPrompt(existingContent string, referenceOverride string) string {
	var b strings.Builder

	b.WriteString("You are a skill authoring assistant. Your job is to help write and refine SKILL.md files for AI agent skills.\n\n")
	b.WriteString(formatSpec)
	b.WriteString("\n\n")

	ref := referenceSkill
	if referenceOverride != "" {
		ref = referenceOverride
	}
	if ref != "" {
		b.WriteString("## Reference Example\n\nHere is a well-formed skill for reference:\n\n```markdown\n")
		b.WriteString(ref)
		b.WriteString("\n```\n\n")
	}

	if existingContent != "" {
		b.WriteString("## Current Skill Content\n\nThe skill currently contains:\n\n```markdown\n")
		b.WriteString(existingContent)
		b.WriteString("\n```\n\n")
		b.WriteString("Help the user refine and improve this skill. Output the complete updated SKILL.md when done.\n")
	} else {
		b.WriteString("Help the user create a new skill from scratch. Output the complete SKILL.md when done.\n")
	}

	fmt.Fprintf(&b, "\nWhen outputting the final SKILL.md, output ONLY the file content (frontmatter + body), no surrounding explanation or code fences.\n")

	return b.String()
}
```

- [ ] **Step 4: Implement LLM package — CLI integration**

```go
// internal/llm/llm.go
package llm

import (
	"fmt"
	"os/exec"
)

// FindCLI checks if the given LLM CLI command is available on PATH.
func FindCLI(command string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("%s CLI not found. Install it or configure a different CLI: coach config set llm-cli <command>", command)
	}
	return path, nil
}

// BuildSingleShotArgs returns the args for a non-interactive single-shot invocation.
// For claude CLI: claude --print -p "prompt" --system-prompt "system"
func BuildSingleShotArgs(cliName, systemPrompt, userPrompt string) []string {
	return []string{
		"--print",
		"-p", userPrompt,
		"--system-prompt", systemPrompt,
	}
}

// BuildInteractiveArgs returns the args for an interactive session.
// For claude CLI: claude --system-prompt "system"
func BuildInteractiveArgs(cliName, systemPrompt string) []string {
	return []string{
		"--system-prompt", systemPrompt,
	}
}
```

- [ ] **Step 5: Copy reference skill for embedding**

The `//go:embed reference_skill.md` directive needs the file next to the Go source:

```bash
cp skill-coach/SKILL.md internal/llm/reference_skill.md
```

- [ ] **Step 6: Run LLM package tests**

Run: `cd /Users/dylan/vault/coach && go test ./internal/llm/ -v`
Expected: PASS

- [ ] **Step 7: Implement generate command**

```go
// cmd/generate.go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/llm"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var (
	generatePrompt string
	generateGlobal bool
	generateLocal  bool
	generateCli    string
)

var generateCmd = &cobra.Command{
	Use:   "generate <skill-name>",
	Short: "Author a skill with LLM assistance",
	Long: `Generate or refine a skill's content using an LLM.

Interactive mode (default) opens a conversation with the LLM to author the skill.
Single-shot mode uses a prompt to generate content in one pass.

If the skill doesn't exist yet, it will be created.

Examples:
  coach generate code-reviewer                              # Interactive: chat with LLM to author the skill
  coach generate code-reviewer -p "help review Go code"     # Single-shot: generate from a prompt
  coach generate new-skill -g                               # Create and author a new global skill
  coach generate my-skill --cli codex                       # Use a different LLM CLI`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	skillName := args[0]

	// Determine CLI to use
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cliName := cfg.LLMCli
	if generateCli != "" {
		cliName = generateCli
	}

	// Check CLI is available
	cliPath, err := llm.FindCLI(cliName)
	if err != nil {
		return err
	}

	// Resolve scope
	scope := resolve.ScopeAny
	if generateGlobal {
		scope = resolve.ScopeGlobal
	} else if generateLocal {
		scope = resolve.ScopeLocal
	}

	coachDir, err := config.DefaultCoachDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         cwd,
	}

	// Try to resolve existing skill, or create new one
	var skillPath string
	var existingContent string

	result, err := r.Resolve(skillName, scope)
	if err != nil {
		// Skill doesn't exist — create directory
		createScope := scope
		if createScope == resolve.ScopeAny {
			// Default to global if not in a project, local if in one
			if cfg.DefaultScope == "local" {
				createScope = resolve.ScopeLocal
			} else {
				createScope = resolve.ScopeGlobal
			}
		}
		targetDir := r.TargetDir(skillName, createScope)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("creating skill directory: %w", err)
		}
		skillPath = filepath.Join(targetDir, "SKILL.md")
		// Write minimal placeholder so the file exists
		placeholder := fmt.Sprintf("---\nname: %s\ndescription: \n---\n", skillName)
		if err := os.WriteFile(skillPath, []byte(placeholder), 0o644); err != nil {
			return fmt.Errorf("writing placeholder: %w", err)
		}
		scopeLabel := "global"
		if createScope == resolve.ScopeLocal {
			scopeLabel = "local"
		}
		fmt.Printf("Creating new %s skill: %s\n", scopeLabel, skillName)
	} else {
		skillPath = result.Path
		data, err := os.ReadFile(result.Path)
		if err != nil {
			return fmt.Errorf("reading existing skill: %w", err)
		}
		existingContent = string(data)
		fmt.Printf("Editing existing skill: %s (%s)\n", skillName, result.Path)
	}

	// Build system prompt
	systemPrompt := llm.BuildSystemPrompt(existingContent, "")

	if generatePrompt != "" {
		// Single-shot mode
		return runSingleShot(cliPath, cliName, systemPrompt, generatePrompt, skillPath, skillName)
	}
	// Interactive mode
	return runInteractive(cliPath, cliName, systemPrompt, skillPath, skillName)
}

func runSingleShot(cliPath, cliName, systemPrompt, userPrompt, skillPath, skillName string) error {
	cliArgs := llm.BuildSingleShotArgs(cliName, systemPrompt, userPrompt)
	cliCmd := exec.Command(cliPath, cliArgs...)
	cliCmd.Stderr = os.Stderr

	output, err := cliCmd.Output()
	if err != nil {
		return fmt.Errorf("LLM CLI failed: %w", err)
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		return fmt.Errorf("LLM returned empty output")
	}

	// Show diff
	existing, _ := os.ReadFile(skillPath)
	if len(existing) > 0 {
		fmt.Println("\n--- Changes ---")
		fmt.Println(content)
		fmt.Println("--- End ---")

		fmt.Print("\nAccept changes? [Y/n] ")
		var response string
		fmt.Scanln(&response)
		if response != "" && response != "y" && response != "Y" {
			fmt.Println("Changes discarded.")
			return nil
		}
	}

	if err := os.WriteFile(skillPath, []byte(content+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing skill: %w", err)
	}

	// Lint
	return lintAfterGenerate(filepath.Dir(skillPath), skillName)
}

func runInteractive(cliPath, cliName, systemPrompt, skillPath, skillName string) error {
	cliArgs := llm.BuildInteractiveArgs(cliName, systemPrompt)
	cliCmd := exec.Command(cliPath, cliArgs...)
	cliCmd.Stdin = os.Stdin
	cliCmd.Stdout = os.Stdout
	cliCmd.Stderr = os.Stderr

	fmt.Printf("Starting interactive session with %s...\n", cliName)
	fmt.Println("Describe what your skill should do. The LLM will help you author it.")
	fmt.Println()

	if err := cliCmd.Run(); err != nil {
		return fmt.Errorf("LLM CLI session ended with error: %w", err)
	}

	// After interactive session, check if skill was updated
	// (The user may have asked the LLM to write to the file directly)
	return lintAfterGenerate(filepath.Dir(skillPath), skillName)
}

func lintAfterGenerate(skillDir, skillName string) error {
	s, err := skill.Parse(skillDir)
	if err != nil {
		fmt.Printf("\n%s Parse error: %s\n", ui.ErrorStyle.Render("✗"), err)
		fmt.Println("Run 'coach edit " + skillName + "' to fix manually.")
		return nil
	}

	errors := skill.Validate(s)
	if len(errors) > 0 {
		fmt.Printf("\n%s Validation issues:\n", ui.ErrorStyle.Render("✗"))
		for _, e := range errors {
			fmt.Printf("  • %s\n", e)
		}
		fmt.Println("Run 'coach edit " + skillName + "' to fix manually.")
		return nil
	}

	fmt.Printf("\n%s Skill %q passes validation.\n", ui.SuccessStyle.Render("✓"), skillName)
	return nil
}

func init() {
	generateCmd.Flags().StringVarP(&generatePrompt, "prompt", "p", "", "Single-shot mode: generate from this prompt")
	generateCmd.Flags().BoolVarP(&generateGlobal, "global", "g", false, "Create/edit in global skills")
	generateCmd.Flags().BoolVarP(&generateLocal, "local", "l", false, "Create/edit in local project skills")
	generateCmd.Flags().StringVar(&generateCli, "cli", "", "Override LLM CLI command")
	rootCmd.AddCommand(generateCmd)
}
```

- [ ] **Step 8: Build and smoke test**

Run: `cd /Users/dylan/vault/coach && go build . && ./coach generate --help`
Expected: Help text displays with examples and flags

- [ ] **Step 9: Commit**

```bash
git add internal/llm/ cmd/generate.go
git commit -m "feat: add coach generate command with LLM-assisted authoring"
```

---

### Task 8: `coach sync` Command

**Files:**
- Create: `cmd/sync.go`

- [ ] **Step 1: Implement sync command**

```go
// cmd/sync.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/distribute"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var (
	syncGlobal bool
	syncLocal  bool
	syncDryRun bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Distribute skills to configured agents",
	Long: `Symlink all managed skills to configured agent directories.

Ensures every skill in your local and global coach directories is
symlinked into the skill directories of your configured agents.

Examples:
  coach sync                # Symlink all skills to configured agents
  coach sync --dry-run      # Preview what would be linked
  coach sync -g             # Sync global skills only`,
	RunE: runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.DistributeTo) == 0 {
		return fmt.Errorf("no distribution targets configured. Run: coach config set distribute-to claude,cursor")
	}

	coachDir, err := config.DefaultCoachDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	scope := resolve.ScopeAny
	if syncGlobal {
		scope = resolve.ScopeGlobal
	} else if syncLocal {
		scope = resolve.ScopeLocal
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         cwd,
	}

	skills, err := r.List(scope)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("No managed skills found.")
		return nil
	}

	// Detect agents and filter to configured targets
	detected, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}
	targets := distribute.FilterAgentsByNames(detected, cfg.DistributeTo)
	if len(targets) == 0 {
		return fmt.Errorf("none of the configured agents (%v) were detected on this system", cfg.DistributeTo)
	}

	if syncDryRun {
		fmt.Println("Dry run — no changes will be made:\n")
	}

	totalCreated := 0
	totalUpToDate := 0

	for _, sk := range skills {
		if syncDryRun {
			for _, t := range targets {
				if !t.Installed {
					continue
				}
				linkPath := filepath.Join(t.SkillDir, sk.Name)
				fmt.Printf("  %s -> %s (%s)\n", linkPath, sk.Dir, t.Config.Name)
			}
			continue
		}

		results, err := distribute.Distribute(sk.Dir, sk.Name, targets)
		if err != nil {
			return fmt.Errorf("distributing %s: %w", sk.Name, err)
		}

		for _, dr := range results {
			switch dr.Status {
			case distribute.StatusCreated:
				fmt.Printf("  %s Linked %s -> %s\n", ui.SuccessStyle.Render("✓"), sk.Name, dr.Agent)
				totalCreated++
			case distribute.StatusUpdated:
				fmt.Printf("  %s Updated %s -> %s\n", ui.SuccessStyle.Render("✓"), sk.Name, dr.Agent)
				totalCreated++
			case distribute.StatusUpToDate:
				totalUpToDate++
			case distribute.StatusSkipped:
				// silent
			}
		}
	}

	if !syncDryRun {
		fmt.Printf("\nSync complete: %d linked, %d already up-to-date.\n", totalCreated, totalUpToDate)
	}

	return nil
}

func init() {
	syncCmd.Flags().BoolVarP(&syncGlobal, "global", "g", false, "Sync global skills only")
	syncCmd.Flags().BoolVarP(&syncLocal, "local", "l", false, "Sync local skills only")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview without making changes")
	rootCmd.AddCommand(syncCmd)
}
```

- [ ] **Step 2: Build and verify**

Run: `cd /Users/dylan/vault/coach && go build .`
Expected: Compiles without errors

- [ ] **Step 3: Commit**

```bash
git add cmd/sync.go
git commit -m "feat: add coach sync command for skill distribution"
```

---

### Task 9: Update `coach init` for Storage Model

**Files:**
- Modify: `cmd/init.go`

- [ ] **Step 1: Add scope flags to init**

Add `--global` / `--local` flags to `coach init skill` and update the output directory logic. Currently `runInitSkill` creates the skill in the current directory. Change it to:

1. If `--local` flag: create in `.coach/skills/<name>/` (relative to project root or cwd)
2. If `--global` flag: create in `~/.coach/skills/<name>/`
3. If neither: use `default_scope` from config (which defaults to `global`)

In `cmd/init.go`, add flag variables:

```go
var (
	initGlobal bool
	initLocal  bool
)
```

In the `init()` function, add:

```go
initSkillCmd.Flags().BoolVarP(&initGlobal, "global", "g", false, "Create in global skills (~/.coach/skills/)")
initSkillCmd.Flags().BoolVarP(&initLocal, "local", "l", false, "Create in local project skills (.coach/skills/)")
```

In `runInitSkill`, after the form completes and `skillName` is determined, replace the directory creation logic:

```go
cfg, err := config.Load()
if err != nil {
    return err
}

coachDir, err := config.DefaultCoachDir()
if err != nil {
    return err
}

var skillDir string
var scopeLabel string
switch {
case initLocal:
    skillDir = filepath.Join(".coach", "skills", skillName)
    scopeLabel = "local"
case initGlobal:
    skillDir = filepath.Join(coachDir, "skills", skillName)
    scopeLabel = "global"
default:
    if cfg.DefaultScope == "local" {
        skillDir = filepath.Join(".coach", "skills", skillName)
        scopeLabel = "local"
    } else {
        skillDir = filepath.Join(coachDir, "skills", skillName)
        scopeLabel = "global"
    }
}
```

Update the success message to show scope and suggest next steps:

```go
fmt.Printf("\n%s Created %s skill: %s\n", ui.SuccessStyle.Render("✓"), scopeLabel, skillDir)
fmt.Println("\nNext steps:")
fmt.Printf("  coach edit %s          # Write skill content in your editor\n", skillName)
fmt.Printf("  coach generate %s      # Or use AI to author it\n", skillName)
fmt.Printf("  coach lint %s          # Validate the skill\n", skillDir)
```

- [ ] **Step 2: Build and smoke test**

Run: `cd /Users/dylan/vault/coach && go build . && ./coach init --help`
Expected: Shows `--global` and `--local` flags

- [ ] **Step 3: Commit**

```bash
git add cmd/init.go
git commit -m "feat(init): add global/local scope flags and updated next-steps text"
```

---

### Task 10: Update Help Text Across All Commands

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/lint.go`
- Modify: `cmd/scan.go` (if exists)

- [ ] **Step 1: Update root help template**

In `cmd/root.go`, update the `customHelp` function to include the new commands and updated getting started flow:

```go
func customHelp(cmd *cobra.Command, args []string) {
    // ... existing styled header ...

    sections := []struct {
        title    string
        commands []struct{ name, desc string }
    }{
        {
            title: "Authoring",
            commands: []struct{ name, desc string }{
                {"init skill", "Scaffold a new skill"},
                {"edit <name>", "Open skill in $EDITOR with lint-on-close"},
                {"generate <name>", "Author or refine a skill with AI assistance"},
            },
        },
        {
            title: "Analysis",
            commands: []struct{ name, desc string }{
                {"lint <path>", "Validate skill spec compliance and security"},
                {"scan <path>", "Deep security analysis of a skill"},
                {"preview <path>", "Render SKILL.md in terminal"},
            },
        },
        {
            title: "Management",
            commands: []struct{ name, desc string }{
                {"install <source>", "Fetch and install skills from a source"},
                {"sync", "Distribute skills to configured agents"},
                {"config set/get", "Manage coach configuration"},
                {"status", "Show agents and installed skills"},
                {"update-rules", "Fetch latest security patterns"},
            },
        },
    }

    // ... render sections ...

    // Updated getting started
    fmt.Println(ui.HeadingStyle.Render("Getting Started"))
    fmt.Println("  1. coach init skill                         Create a new skill")
    fmt.Println("  2. coach edit <name>                        Write the skill content")
    fmt.Println("     coach generate <name>                    Or use AI to author it")
    fmt.Println("  3. coach lint <path>                        Validate the skill")
    fmt.Println("  4. coach config set distribute-to claude    Configure distribution")
    fmt.Println("  5. coach sync                               Symlink skills to agents")
    fmt.Println()
}
```

- [ ] **Step 2: Update lint and scan help to differentiate them**

In `cmd/lint.go`, update the Long description:

```go
Long: `Validate a skill against the SKILL.md specification and run security checks.

Lint checks for:
  - Required frontmatter fields (name, description)
  - Field format and length constraints
  - Body content presence
  - Common security patterns (prompt injection, dangerous commands)

Use 'coach scan' for deeper security analysis with the full pattern database.

Examples:
  coach lint .                    # Lint skill in current directory
  coach lint ./my-skill           # Lint a specific skill
  coach lint ./my-skill --json    # Output results as JSON`,
```

In `cmd/scan.go` (if it has a separate file), update similarly to differentiate:

```go
Long: `Deep security analysis of a skill using the full pattern database.

Scan performs thorough analysis including:
  - Prompt injection detection across all files
  - Script analysis for dangerous shell patterns
  - Quality checks (missing allowed-tools, weak descriptions)
  - Risk scoring with severity-weighted findings

Use 'coach lint' for quick spec validation during development.

Examples:
  coach scan ./my-skill           # Full security scan
  coach scan ./my-skill --json    # Output results as JSON`,
```

- [ ] **Step 3: Build and verify help output**

Run: `cd /Users/dylan/vault/coach && go build . && ./coach --help`
Expected: Updated categories with new commands, updated getting started section

Run: `cd /Users/dylan/vault/coach && ./coach lint --help && ./coach scan --help`
Expected: Clear differentiation between the two commands

- [ ] **Step 4: Commit**

```bash
git add cmd/root.go cmd/lint.go cmd/scan.go
git commit -m "docs: update help text with new commands and clearer examples"
```

---

### Task 11: Integration Test — Full Workflow

**Files:**
- Create: `cmd/integration_test.go`

- [ ] **Step 1: Write integration test for the full workflow**

```go
// cmd/integration_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/distribute"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/pkg"
)

func TestFullWorkflow(t *testing.T) {
	// Setup: fake home dir with coach structure
	homeDir := t.TempDir()
	globalSkillsDir := filepath.Join(homeDir, ".coach", "skills")
	agentSkillDir := filepath.Join(homeDir, ".claude", "skills")
	os.MkdirAll(globalSkillsDir, 0o755)
	os.MkdirAll(agentSkillDir, 0o755)

	// Step 1: Create a skill (simulating coach init)
	skillName := "test-reviewer"
	skillDir := filepath.Join(globalSkillsDir, skillName)
	os.MkdirAll(skillDir, 0o755)

	content := `---
name: test-reviewer
description: Reviews test quality and coverage
license: MIT
allowed-tools:
  - Read
  - Grep
---

# Test Reviewer

## When to Use

Use when the user asks to review tests or check test coverage.

## Instructions

1. Read the test files in the project
2. Check for common testing anti-patterns
3. Suggest improvements

## Constraints

- Do not modify test files without asking
- Do not delete existing tests
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644)

	// Step 2: Resolve the skill
	r := resolve.Resolver{GlobalSkillsDir: globalSkillsDir, WorkDir: homeDir}
	result, err := r.Resolve(skillName, resolve.ScopeAny)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if result.Scope != resolve.ScopeGlobal {
		t.Errorf("scope = %v, want ScopeGlobal", result.Scope)
	}

	// Step 3: Parse and validate
	s, err := skill.Parse(result.Dir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	errors := skill.Validate(s)
	if len(errors) > 0 {
		t.Errorf("Validate returned errors: %v", errors)
	}

	// Step 4: Distribute
	agents := []pkg.DetectedAgent{
		{
			Config:    pkg.AgentConfig{Name: "claude-code", SkillDir: agentSkillDir + "/"},
			Installed: true,
			SkillDir:  agentSkillDir,
		},
	}

	results, err := distribute.Distribute(result.Dir, skillName, agents)
	if err != nil {
		t.Fatalf("Distribute failed: %v", err)
	}
	if results[0].Status != distribute.StatusCreated {
		t.Errorf("status = %v, want StatusCreated", results[0].Status)
	}

	// Verify symlink exists and points to right place
	linkPath := filepath.Join(agentSkillDir, skillName)
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if target != result.Dir {
		t.Errorf("symlink target = %q, want %q", target, result.Dir)
	}

	// Step 5: Config round-trip
	configPath := filepath.Join(homeDir, ".coach", "config.yaml")
	cfg := config.Config{
		DistributeTo: []string{"claude-code"},
		LLMCli:       "claude",
		DefaultScope: "global",
	}
	if err := config.SaveTo(cfg, configPath); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}
	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}
	if loaded.LLMCli != "claude" {
		t.Errorf("LLMCli = %q, want %q", loaded.LLMCli, "claude")
	}
}
```

- [ ] **Step 2: Run integration test**

Run: `cd /Users/dylan/vault/coach && go test ./cmd/ -v -run TestFullWorkflow`
Expected: PASS

- [ ] **Step 3: Run all tests**

Run: `cd /Users/dylan/vault/coach && go test ./... -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/integration_test.go
git commit -m "test: add integration test for full skill workflow"
```

---

### Task 12: Final Build and Lint

- [ ] **Step 1: Build the binary**

Run: `cd /Users/dylan/vault/coach && go build -o coach .`
Expected: Compiles without errors

- [ ] **Step 2: Run go vet**

Run: `cd /Users/dylan/vault/coach && go vet ./...`
Expected: No issues

- [ ] **Step 3: Run golangci-lint if available**

Run: `cd /Users/dylan/vault/coach && golangci-lint run ./...`
Expected: No new issues (existing issues are acceptable)

- [ ] **Step 4: Verify all help text**

Run these and review output:
```bash
./coach --help
./coach edit --help
./coach generate --help
./coach config --help
./coach config set --help
./coach sync --help
./coach init skill --help
./coach lint --help
```

Expected: All commands show clear descriptions and examples

- [ ] **Step 5: Final commit**

```bash
git commit --allow-empty -m "feat: complete v0.2 skill editing and authoring milestone"
```
