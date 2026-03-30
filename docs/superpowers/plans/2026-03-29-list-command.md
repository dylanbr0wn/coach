# `coach list` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `coach list` command that shows installed skills per agent with name, description, path, and vetted status.

**Architecture:** Extract `listSkillDirs` from `cmd/status.go` into `internal/skill` as a shared helper. Add a `Key` field to `DetectedAgent` so agents can be filtered by registry key. New `cmd/list.go` command uses existing agent detection, skill parsing, and provenance loading to render a grouped table or JSON.

**Tech Stack:** Go, Cobra, lipgloss (via `internal/ui`)

---

### Task 1: Add `Key` field to `DetectedAgent`

`DetectedAgent` currently doesn't store the registry key (e.g., `"claude-code"`). The `--agent` flag needs it for filtering.

**Files:**
- Modify: `pkg/types.go:134-138`
- Modify: `internal/agent/detect.go:34-46`
- Modify: `internal/agent/detect_test.go`

- [ ] **Step 1: Write failing test for Key field**

Add to `internal/agent/detect_test.go`:

```go
func TestDetectAgents_HasKey(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents, err := DetectAgentsInHome(tmpHome)
	if err != nil {
		t.Fatalf("DetectAgentsInHome() error: %v", err)
	}

	for _, a := range agents {
		if a.Key == "" {
			t.Errorf("agent %q has empty Key", a.Config.Name)
		}
	}

	// Check specific key mapping
	for _, a := range agents {
		if a.Config.Name == "Claude Code" && a.Key != "claude-code" {
			t.Errorf("Claude Code Key = %q, want %q", a.Key, "claude-code")
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/ -run TestDetectAgents_HasKey -v`
Expected: FAIL — `a.Key` does not exist on `DetectedAgent`

- [ ] **Step 3: Add Key field to DetectedAgent and populate it**

In `pkg/types.go`, add `Key` to the `DetectedAgent` struct:

```go
type DetectedAgent struct {
	Key       string // Registry key, e.g. "claude-code"
	Config    AgentConfig
	Installed bool   // Whether the agent's directory exists
	SkillDir  string // Resolved absolute path to skill directory
}
```

In `internal/agent/detect.go`, update `DetectAgentsInHome` to store the key:

```go
for key, agentCfg := range reg.Agents {
	resolvedDir := resolveHomePath(agentCfg.SkillDir, home)
	installed := dirExists(resolvedDir)

	detected = append(detected, pkg.DetectedAgent{
		Key:       key,
		Config:    agentCfg,
		Installed: installed,
		SkillDir:  resolvedDir,
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/agent/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/types.go internal/agent/detect.go internal/agent/detect_test.go
git commit -m "feat: add Key field to DetectedAgent for agent filtering"
```

---

### Task 2: Extract `ListSkillDirs` to `internal/skill`

`listSkillDirs` in `cmd/status.go:90-108` is unexported. Both `status` and the new `list` command need it.

**Files:**
- Modify: `internal/skill/skill.go`
- Modify: `cmd/status.go:90-108`
- Test: `internal/skill/skill_test.go`

- [ ] **Step 1: Write failing test for ListSkillDirs**

Add to `internal/skill/skill_test.go`:

```go
func TestListSkillDirs(t *testing.T) {
	dir := t.TempDir()

	// Create two skill directories with SKILL.md
	for _, name := range []string{"skill-a", "skill-b"} {
		skillDir := filepath.Join(dir, name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: test\n---\nBody"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a directory without SKILL.md (should be excluded)
	if err := os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	names := ListSkillDirs(dir)
	if len(names) != 2 {
		t.Fatalf("ListSkillDirs() returned %d names, want 2: %v", len(names), names)
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["skill-a"] || !nameSet["skill-b"] {
		t.Errorf("expected skill-a and skill-b, got %v", names)
	}
}

func TestListSkillDirs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := ListSkillDirs(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 names for empty dir, got %d", len(names))
	}
}

func TestListSkillDirs_NonexistentDir(t *testing.T) {
	names := ListSkillDirs("/nonexistent/path")
	if len(names) != 0 {
		t.Errorf("expected 0 names for nonexistent dir, got %d", len(names))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/skill/ -run TestListSkillDirs -v`
Expected: FAIL — `ListSkillDirs` does not exist

- [ ] **Step 3: Move function to internal/skill and export it**

Add to the bottom of `internal/skill/skill.go`:

```go
// ListSkillDirs returns the names of subdirectories in dir that contain a SKILL.md file.
// It also detects a flat layout where SKILL.md is directly in dir.
func ListSkillDirs(dir string) []string {
	var names []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return names
	}
	for _, e := range entries {
		if e.IsDir() {
			skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				names = append(names, e.Name())
			}
		}
		if !e.IsDir() && strings.EqualFold(e.Name(), "SKILL.md") {
			names = append(names, filepath.Base(dir))
		}
	}
	return names
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/skill/ -v`
Expected: All tests PASS

- [ ] **Step 5: Update status.go to use the shared function**

In `cmd/status.go`, replace the call to `listSkillDirs` with `skill.ListSkillDirs`, add the import `"github.com/dylanbr0wn/coach/internal/skill"`, and delete the local `listSkillDirs` function (lines 90-108).

Change line 58:
```go
skillNames := skill.ListSkillDirs(a.SkillDir)
```

Remove the `path/filepath` import from `cmd/status.go` since it's no longer used there.

- [ ] **Step 6: Run all tests to verify nothing broke**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add internal/skill/skill.go internal/skill/skill_test.go cmd/status.go
git commit -m "refactor: extract ListSkillDirs to internal/skill for reuse"
```

---

### Task 3: Implement `coach list` command (table output)

**Files:**
- Create: `cmd/list.go`
- Create: `cmd/list_test.go`

- [ ] **Step 1: Write test for list command with table output**

Create `cmd/list_test.go`:

```go
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestListCommand_ShowsSkills(t *testing.T) {
	// Set up a temp home with one agent and two skills
	tmpHome := t.TempDir()
	skillsDir := filepath.Join(tmpHome, ".claude", "skills")

	for _, name := range []string{"alpha-skill", "beta-skill"} {
		dir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\nname: " + name + "\ndescription: A test skill\n---\nBody"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Set up coach dir with empty provenance
	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "", "table")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("alpha-skill")) {
		t.Errorf("expected output to contain 'alpha-skill', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("beta-skill")) {
		t.Errorf("expected output to contain 'beta-skill', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Claude Code")) {
		t.Errorf("expected output to contain 'Claude Code', got:\n%s", output)
	}
}

func TestListCommand_NoAgents(t *testing.T) {
	tmpHome := t.TempDir()
	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "", "table")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("No agents detected")) {
		t.Errorf("expected 'No agents detected' message, got:\n%s", output)
	}
}

func TestListCommand_AgentFilter(t *testing.T) {
	tmpHome := t.TempDir()
	skillsDir := filepath.Join(tmpHome, ".claude", "skills")
	dir := filepath.Join(skillsDir, "my-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: my-skill\ndescription: A test skill\n---\nBody"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "claude-code", "table")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("my-skill")) {
		t.Errorf("expected output to contain 'my-skill', got:\n%s", output)
	}
}

func TestListCommand_InvalidAgent(t *testing.T) {
	tmpHome := t.TempDir()
	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "nonexistent-agent", "table")
	if err == nil {
		t.Fatal("expected error for invalid agent, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestListCommand -v`
Expected: FAIL — `runListWithHome` does not exist

- [ ] **Step 3: Implement the list command**

Create `cmd/list.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/dylanbr0wn/coach/pkg"
	"github.com/spf13/cobra"
)

var (
	listAgent  string
	listFormat string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills per agent",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listAgent, "agent", "", "Filter to a specific agent (e.g. claude-code, cursor)")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format: table or json")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	return runListWithHome(os.Stdout, "", config.DefaultCoachDir(), listAgent, listFormat)
}

// runListWithHome is the testable core. If home is empty, it uses os.UserHomeDir.
func runListWithHome(w io.Writer, home, coachDir, agentFilter, format string) error {
	var agents []pkg.DetectedAgent
	var err error
	if home != "" {
		agents, err = agent.DetectAgentsInHome(home)
	} else {
		agents, err = agent.DetectAgents("")
	}
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}

	installed := agent.InstalledAgents(agents)

	// Filter by agent key if requested
	if agentFilter != "" {
		var filtered []pkg.DetectedAgent
		for _, a := range installed {
			if a.Key == agentFilter {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) == 0 {
			var keys []string
			for _, a := range agents {
				keys = append(keys, a.Key)
			}
			return fmt.Errorf("unknown agent %q (available: %v)", agentFilter, keys)
		}
		installed = filtered
	}

	if len(installed) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.WarningStyle.Render("  No agents detected."))
		fmt.Fprintln(w)
		return nil
	}

	provenance, _ := registry.LoadProvenance(coachDir)
	provenanceMap := make(map[string]bool)
	for _, s := range provenance.Skills {
		provenanceMap[s.Name] = true
	}

	type skillInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Path        string `json:"path"`
		Vetted      bool   `json:"vetted"`
	}

	type agentGroup struct {
		Agent    string      `json:"agent"`
		SkillDir string      `json:"skill_dir"`
		Skills   []skillInfo `json:"skills"`
	}

	var groups []agentGroup

	for _, a := range installed {
		group := agentGroup{
			Agent:    a.Config.Name,
			SkillDir: a.SkillDir,
		}

		skillNames := skill.ListSkillDirs(a.SkillDir)
		for _, name := range skillNames {
			si := skillInfo{
				Name:   name,
				Path:   a.SkillDir + name + "/",
				Vetted: provenanceMap[name],
			}

			// Try to parse for description; gracefully handle errors
			parsed, err := skill.Parse(a.SkillDir + name)
			if err == nil {
				si.Description = parsed.Description
			} else {
				si.Description = "(parse error)"
			}

			group.Skills = append(group.Skills, si)
		}

		groups = append(groups, group)
	}

	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(groups)
	}

	// Table output
	for i, group := range groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.HeadingStyle.Render("  "+group.Agent)+" "+ui.DimStyle.Render("("+group.SkillDir+")"))
		fmt.Fprintln(w)

		if len(group.Skills) == 0 {
			fmt.Fprintln(w, ui.DimStyle.Render("  No skills installed"))
			continue
		}

		var rows []ui.TableRow
		for _, s := range group.Skills {
			desc := s.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			vetted := ui.SuccessStyle.Render("✓")
			if !s.Vetted {
				vetted = ui.WarningStyle.Render("✗")
			}

			rows = append(rows, ui.TableRow{
				Cells: []string{s.Name, desc, s.Path, vetted},
			})
		}

		fmt.Fprint(w, ui.RenderTable(
			[]string{"Name", "Description", "Path", "Vetted"},
			rows,
		))
	}

	fmt.Fprintln(w)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run TestListCommand -v`
Expected: All tests PASS

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/list.go cmd/list_test.go
git commit -m "feat: add coach list command with table and JSON output"
```

---

### Task 4: Add `list` to help template and JSON output test

**Files:**
- Modify: `cmd/root.go:55-58`
- Modify: `cmd/list_test.go`

- [ ] **Step 1: Write test for JSON output**

Add to `cmd/list_test.go`:

```go
func TestListCommand_JSONOutput(t *testing.T) {
	tmpHome := t.TempDir()
	skillsDir := filepath.Join(tmpHome, ".claude", "skills")
	dir := filepath.Join(skillsDir, "json-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: json-skill\ndescription: A JSON test skill\n---\nBody"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	coachDir := filepath.Join(tmpHome, ".coach")
	if err := os.MkdirAll(coachDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runListWithHome(&buf, tmpHome, coachDir, "claude-code", "json")
	if err != nil {
		t.Fatalf("runListWithHome() error: %v", err)
	}

	var result []struct {
		Agent    string `json:"agent"`
		SkillDir string `json:"skill_dir"`
		Skills   []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Path        string `json:"path"`
			Vetted      bool   `json:"vetted"`
		} `json:"skills"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal error: %v\nOutput: %s", err, buf.String())
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 agent group, got %d", len(result))
	}
	if len(result[0].Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result[0].Skills))
	}
	if result[0].Skills[0].Name != "json-skill" {
		t.Errorf("skill name = %q, want %q", result[0].Skills[0].Name, "json-skill")
	}
	if result[0].Skills[0].Description != "A JSON test skill" {
		t.Errorf("skill description = %q, want %q", result[0].Skills[0].Description, "A JSON test skill")
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./cmd/ -run TestListCommand_JSONOutput -v`
Expected: PASS (implementation already supports JSON)

- [ ] **Step 3: Add `list` to the Management group in help template**

In `cmd/root.go`, add the `list` entry to the Management section (after the `status` line):

```go
fmt.Fprintf(&b, "\n%s\n", h("Management"))
fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "install"))
fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "list"))
fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "status"))
fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "update-rules"))
```

- [ ] **Step 4: Run all tests and verify build**

Run: `go test ./... && go build .`
Expected: All tests PASS, binary builds cleanly

- [ ] **Step 5: Commit**

```bash
git add cmd/list_test.go cmd/root.go
git commit -m "feat: add list to help template and add JSON output test"
```

---

### Task 5: Run lint and manual smoke test

**Files:** None (verification only)

- [ ] **Step 1: Run linter**

Run: `golangci-lint run ./...`
Expected: No errors

- [ ] **Step 2: Build and run smoke tests**

Run:
```bash
go build -o coach .
./coach list
./coach list --agent claude-code
./coach list --format json
./coach list --agent nonexistent
./coach --help
```

Verify:
- `coach list` shows skills grouped by agent with table output
- `--agent claude-code` filters to Claude Code only
- `--format json` produces valid JSON
- `--agent nonexistent` produces an error with valid agent keys
- `--help` shows `list` in the Management group

- [ ] **Step 3: Fix any issues found**

Address any lint errors or output formatting issues.

- [ ] **Step 4: Final commit if fixes were needed**

```bash
git add -A
git commit -m "fix: address lint and formatting issues in list command"
```
