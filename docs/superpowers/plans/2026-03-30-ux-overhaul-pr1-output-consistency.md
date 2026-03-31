# PR 1: Output Consistency — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Standardize all CLI output through shared helper functions so every command uses consistent formatting patterns.

**Architecture:** Add new helper functions to `internal/ui/` that encode the output patterns (success, error, warning, next-step, spinner). Then audit every command file to replace ad-hoc formatting with these helpers. The existing lipgloss styles and rendering functions (`RenderFindings`, `RenderScanSummary`, etc.) stay as-is.

**Tech Stack:** Go, charmbracelet/lipgloss (existing), charmbracelet/bubbles/spinner (promote from indirect to direct), charmbracelet/bubbletea (promote from indirect to direct)

---

### Task 1: Add `ui.Success`, `ui.Warn`, `ui.Error` helpers

**Files:**
- Create: `internal/ui/helpers.go`
- Create: `internal/ui/helpers_test.go`

- [ ] **Step 1: Write failing tests for Success, Warn, Error**

```go
// internal/ui/helpers_test.go
package ui

import (
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	got := Success("Skill created")
	if !strings.Contains(got, "✓") {
		t.Errorf("Success() missing checkmark icon, got: %q", got)
	}
	if !strings.Contains(got, "Skill created") {
		t.Errorf("Success() missing message, got: %q", got)
	}
}

func TestWarn(t *testing.T) {
	got := Warn("No agents configured")
	if !strings.Contains(got, "⚠") {
		t.Errorf("Warn() missing warning icon, got: %q", got)
	}
	if !strings.Contains(got, "No agents configured") {
		t.Errorf("Warn() missing message, got: %q", got)
	}
}

func TestError(t *testing.T) {
	got := Error("Skill not found", "Run 'coach list' to see available skills")
	if !strings.Contains(got, "✗") {
		t.Errorf("Error() missing X icon, got: %q", got)
	}
	if !strings.Contains(got, "Skill not found") {
		t.Errorf("Error() missing message, got: %q", got)
	}
	if !strings.Contains(got, "→") {
		t.Errorf("Error() missing arrow for suggestion, got: %q", got)
	}
	if !strings.Contains(got, "Run 'coach list'") {
		t.Errorf("Error() missing suggestion, got: %q", got)
	}
}

func TestErrorNoSuggestion(t *testing.T) {
	got := Error("Something broke", "")
	if strings.Contains(got, "→") {
		t.Errorf("Error() with empty suggestion should not contain arrow, got: %q", got)
	}
	if !strings.Contains(got, "✗") {
		t.Errorf("Error() missing X icon, got: %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./internal/ui/ -run 'TestSuccess|TestWarn|TestError' -v`
Expected: compilation error — `Success`, `Warn`, `Error` not defined

- [ ] **Step 3: Implement the helpers**

```go
// internal/ui/helpers.go
package ui

import "fmt"

// Success returns a formatted success message: ✓ msg
func Success(msg string) string {
	return fmt.Sprintf("%s %s", SuccessStyle.Render("✓"), msg)
}

// Warn returns a formatted warning message: ⚠ msg
func Warn(msg string) string {
	return fmt.Sprintf("%s %s", WarningStyle.Render("⚠"), msg)
}

// Error returns a formatted error message with an optional suggestion.
//
//	✗ msg
//	  → suggestion
func Error(msg, suggestion string) string {
	line := fmt.Sprintf("%s %s", ErrorStyle.Render("✗"), msg)
	if suggestion != "" {
		line += fmt.Sprintf("\n  %s %s", InfoStyle.Render("→"), suggestion)
	}
	return line
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./internal/ui/ -run 'TestSuccess|TestWarn|TestError' -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/helpers.go internal/ui/helpers_test.go
git commit -m "feat(ui): add Success, Warn, Error helper functions"
```

---

### Task 2: Add `ui.NextStep` helper

**Files:**
- Modify: `internal/ui/helpers.go`
- Modify: `internal/ui/helpers_test.go`

- [ ] **Step 1: Write failing test**

Append to `internal/ui/helpers_test.go`:

```go
func TestNextStep(t *testing.T) {
	got := NextStep("generate my-skill", "flesh out the skill with an LLM")
	if !strings.Contains(got, "coach generate my-skill") {
		t.Errorf("NextStep() missing command, got: %q", got)
	}
	if !strings.Contains(got, "flesh out the skill") {
		t.Errorf("NextStep() missing description, got: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./internal/ui/ -run TestNextStep -v`
Expected: compilation error — `NextStep` not defined

- [ ] **Step 3: Implement NextStep**

Append to `internal/ui/helpers.go`:

```go
// NextStep returns a dimmed hint for what command to run next.
func NextStep(cmd, desc string) string {
	return DimStyle.Render(fmt.Sprintf("Next: coach %s — %s", cmd, desc))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./internal/ui/ -run TestNextStep -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/helpers.go internal/ui/helpers_test.go
git commit -m "feat(ui): add NextStep helper for post-command hints"
```

---

### Task 3: Add `ui.WithSpinner` helper

**Files:**
- Create: `internal/ui/spinner.go`
- Create: `internal/ui/spinner_test.go`
- Modify: `go.mod` (promote bubbles/bubbletea from indirect to direct)

- [ ] **Step 1: Promote bubbles and bubbletea to direct dependencies**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go get github.com/charmbracelet/bubbles github.com/charmbracelet/bubbletea`

- [ ] **Step 2: Write failing test**

```go
// internal/ui/spinner_test.go
package ui

import (
	"errors"
	"testing"
)

func TestWithSpinnerSuccess(t *testing.T) {
	called := false
	err := WithSpinner("Working...", func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithSpinner returned error: %v", err)
	}
	if !called {
		t.Error("function was not called")
	}
}

func TestWithSpinnerError(t *testing.T) {
	want := errors.New("something failed")
	got := WithSpinner("Working...", func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Errorf("WithSpinner error = %v, want %v", got, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./internal/ui/ -run TestWithSpinner -v`
Expected: compilation error — `WithSpinner` not defined

- [ ] **Step 4: Implement WithSpinner**

```go
// internal/ui/spinner.go
package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg struct{ err error }

type spinnerModel struct {
	spinner spinner.Model
	msg     string
	fn      func() error
	done    bool
	err     error
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		return errMsg{err: m.fn()}
	})
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("  %s %s\n", m.spinner.View(), m.msg)
}

// WithSpinner runs fn while displaying an animated spinner with the given message.
// Returns the error from fn.
func WithSpinner(msg string, fn func() error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = InfoStyle

	m := spinnerModel{
		spinner: s,
		msg:     msg,
		fn:      fn,
	}

	p := tea.NewProgram(m, tea.WithOutput(nil))
	finalModel, err := p.Run()
	if err != nil {
		// Bubbletea error — fall back to running without spinner.
		return fn()
	}

	return finalModel.(spinnerModel).err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./internal/ui/ -run TestWithSpinner -v`
Expected: both tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ui/spinner.go internal/ui/spinner_test.go go.mod go.sum
git commit -m "feat(ui): add WithSpinner helper using bubbles spinner"
```

---

### Task 4: Migrate `edit.go` to use helpers

**Files:**
- Modify: `cmd/edit.go`

Current ad-hoc patterns to replace:
- Line 101: `fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Parse error: %v", err)))` → `ui.Error`
- Line 112: `fmt.Println(ui.ErrorStyle.Render("  ✗ " + issue))` → `ui.Error`
- Line 121: `fmt.Printf("%s %s saved and validated.\n", ui.SuccessStyle.Render("✓"), name)` → `ui.Success`

- [ ] **Step 1: Replace parse error output**

In `cmd/edit.go`, replace line 101:
```go
// Before:
fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Parse error: %v", err)))
// After:
fmt.Println(ui.Error(fmt.Sprintf("Parse error: %v", err), ""))
```

- [ ] **Step 2: Replace validation issues output**

In `cmd/edit.go`, replace lines 110-113:
```go
// Before:
fmt.Println()
for _, issue := range issues {
    fmt.Println(ui.ErrorStyle.Render("  ✗ " + issue))
}
// After:
fmt.Println()
for _, issue := range issues {
    fmt.Println(ui.Error(issue, ""))
}
```

- [ ] **Step 3: Replace success message**

In `cmd/edit.go`, replace line 121:
```go
// Before:
fmt.Printf("%s %s saved and validated.\n", ui.SuccessStyle.Render("✓"), name)
// After:
fmt.Println(ui.Success(fmt.Sprintf("%s saved and validated.", name)))
```

- [ ] **Step 4: Run existing tests to verify nothing broke**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./cmd/ -run TestGetEditor -v && go build ./...`
Expected: PASS + successful build

- [ ] **Step 5: Commit**

```bash
git add cmd/edit.go
git commit -m "refactor(edit): migrate to ui helper functions"
```

---

### Task 5: Migrate `generate.go` to use helpers + add spinner

**Files:**
- Modify: `cmd/generate.go`

Current ad-hoc patterns to replace:
- Line 149: success with `ui.SuccessStyle.Render("✓")` → `ui.Success`
- Lines 152-154: next steps block with `ui.InfoStyle.Render` → `ui.NextStep`
- Line 161: info arrow with `ui.InfoStyle.Render("→")` — keep as-is (it's an intro message, not a next-step hint)
- Line 174: error with `ui.ErrorStyle.Render` → `ui.Error`
- Line 175: suggestion with `ui.InfoStyle.Render` → fold into `ui.Error`
- Lines 182-185: validation errors + suggestion → `ui.Error`
- Line 189: success with `ui.SuccessStyle.Render("✓")` → `ui.Success`
- Lines 191-192: next steps → `ui.NextStep`

- [ ] **Step 1: Add spinner to single-shot LLM call**

In `cmd/generate.go`, replace lines 127-135 in `runSingleShot`:
```go
// Before:
output, err := llm.RunSingleShot(cliPath, systemPrompt, userPrompt)
if err != nil {
    return fmt.Errorf("LLM CLI error: %w", err)
}

result := strings.TrimSpace(string(output))
if result == "" {
    return fmt.Errorf("LLM returned empty output")
}
// After:
var output []byte
if spinErr := ui.WithSpinner("Generating with LLM...", func() error {
    var llmErr error
    output, llmErr = llm.RunSingleShot(cliPath, systemPrompt, userPrompt)
    return llmErr
}); spinErr != nil {
    return fmt.Errorf("LLM CLI error: %w", spinErr)
}

result := strings.TrimSpace(string(output))
if result == "" {
    return fmt.Errorf("LLM returned empty output")
}
```

Note: Interactive mode (`runInteractive`) does NOT get a spinner — the user is actively typing in the LLM session.

- [ ] **Step 2: Migrate `runSingleShot` success + next steps**

In `cmd/generate.go`, replace lines 149-155:
```go
// Before:
fmt.Printf("  %s Skill updated: %s\n", ui.SuccessStyle.Render("✓"), skillName)
fmt.Printf("  Path: %s\n", skillPath)
fmt.Println()
fmt.Printf("  Next steps:\n")
fmt.Printf("    %-36s   Validate all managed skills\n", ui.InfoStyle.Render("coach lint"))
fmt.Printf("    %-36s   Distribute to your agents\n", ui.InfoStyle.Render("coach sync"))
fmt.Println()
// After:
fmt.Println(ui.Success(fmt.Sprintf("Skill updated: %s", skillName)))
fmt.Printf("  Path: %s\n", skillPath)
fmt.Println()
fmt.Println(ui.NextStep("lint "+skillName, "validate before distributing"))
```

- [ ] **Step 3: Migrate `lintAfterGenerate` error messages**

In `cmd/generate.go`, replace lines 174-175:
```go
// Before:
fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("✗ Parse error: %v", parseErr)))
fmt.Printf("  Run %s to fix manually.\n", ui.InfoStyle.Render("coach edit "+skillName))
// After:
fmt.Println(ui.Error(fmt.Sprintf("Parse error: %v", parseErr), fmt.Sprintf("Run 'coach edit %s' to fix manually", skillName)))
```

- [ ] **Step 4: Migrate validation errors in `lintAfterGenerate`**

In `cmd/generate.go`, replace lines 181-185:
```go
// Before:
fmt.Println()
for _, issue := range issues {
    fmt.Println(ui.ErrorStyle.Render("  ✗ " + issue))
}
fmt.Printf("\n  Run %s to fix manually.\n", ui.InfoStyle.Render("coach edit "+skillName))
// After:
fmt.Println()
for _, issue := range issues {
    fmt.Println(ui.Error(issue, ""))
}
fmt.Println(ui.Error("", fmt.Sprintf("Run 'coach edit %s' to fix manually", skillName)))
```

Wait — that's awkward with an empty message. Better approach:

```go
// After:
fmt.Println()
for _, issue := range issues {
    fmt.Println(ui.Error(issue, ""))
}
fmt.Printf("\n  %s %s\n", ui.InfoStyle.Render("→"), fmt.Sprintf("Run 'coach edit %s' to fix manually", skillName))
```

Actually, let's add a `ui.Suggestion` helper for standalone suggestion lines. But that's scope creep. Keep the `InfoStyle` arrow for now — it's a standalone suggestion, not part of an error. The `ui.Error` helper is for error+suggestion pairs.

```go
// After:
fmt.Println()
for _, issue := range issues {
    fmt.Println(ui.Error(issue, ""))
}
fmt.Println()
fmt.Println(ui.Error("Validation failed", fmt.Sprintf("Run 'coach edit %s' to fix manually", skillName)))
```

- [ ] **Step 5: Migrate success + next step in `lintAfterGenerate`**

In `cmd/generate.go`, replace lines 189-193:
```go
// Before:
fmt.Printf("  %s %s validated successfully.\n", ui.SuccessStyle.Render("✓"), skillName)
fmt.Println()
fmt.Printf("  Next steps:\n")
fmt.Printf("    %-36s   Distribute to your agents\n", ui.InfoStyle.Render("coach sync"))
fmt.Println()
// After:
fmt.Println(ui.Success(fmt.Sprintf("%s validated successfully.", skillName)))
fmt.Println()
fmt.Println(ui.NextStep("sync", "distribute to your agents"))
```

- [ ] **Step 6: Verify build succeeds**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./...`
Expected: successful build

- [ ] **Step 7: Commit**

```bash
git add cmd/generate.go
git commit -m "refactor(generate): migrate to ui helpers + add LLM spinner"
```

---

### Task 6: Migrate `init.go` to use helpers

**Files:**
- Modify: `cmd/init.go`

Current ad-hoc patterns:
- Line 183: success with `ui.SuccessStyle.Render("✓")` → `ui.Success`
- Lines 186-191: next steps with `lipgloss.NewStyle().Bold(true)` → `ui.NextStep`

- [ ] **Step 1: Replace success message and next steps block**

In `cmd/init.go`, replace lines 182-192:
```go
// Before:
fmt.Println()
fmt.Printf("  %s Skill created: %s\n", ui.SuccessStyle.Render("✓"), name)
fmt.Printf("  Path: %s\n", dir)
fmt.Println()
boldStyle := lipgloss.NewStyle().Bold(true)
fmt.Printf("  Next steps:\n")
fmt.Printf("    %-36s   Edit the skill\n", boldStyle.Render("coach edit "+name))
fmt.Printf("    %-36s   Author with AI\n", boldStyle.Render("coach generate "+name))
fmt.Printf("    %-36s   Validate all managed skills\n", boldStyle.Render("coach lint"))
fmt.Printf("    %-36s   Distribute to your agents\n", boldStyle.Render("coach sync"))
fmt.Println()
// After:
fmt.Println()
fmt.Println(ui.Success(fmt.Sprintf("Skill created: %s", name)))
fmt.Printf("  Path: %s\n", dir)
fmt.Println()
fmt.Println(ui.NextStep(fmt.Sprintf("edit %s", name), "edit the skill manually"))
fmt.Println(ui.NextStep(fmt.Sprintf("generate %s", name), "author with AI"))
```

- [ ] **Step 2: Remove unused lipgloss import if no longer needed**

Check if `lipgloss` is still used elsewhere in `init.go`. It's imported on line 10 — the only usage was the `boldStyle` on line 186. Remove the import:

```go
// Remove from imports:
"github.com/charmbracelet/lipgloss"
```

- [ ] **Step 3: Verify build succeeds**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./...`
Expected: successful build

- [ ] **Step 4: Commit**

```bash
git add cmd/init.go
git commit -m "refactor(init): migrate to ui helper functions"
```

---

### Task 7: Migrate `install.go` to use helpers

**Files:**
- Modify: `cmd/install.go`

Current ad-hoc patterns:
- Line 71: warning with `ui.WarningStyle.Render("?")` — keep as-is (it's a special parse-error indicator, not a standard warning)
- Line 74: `ui.SuccessStyle.Render("•")` — keep as-is (bullet point for list items)
- Line 125: error with `ui.ErrorStyle.Render("✗")` → `ui.Error`
- Line 134: error with `ui.ErrorStyle.Render(...)` → `ui.Error`
- Line 148: dim skipped message — keep as-is
- Line 158: error with `ui.ErrorStyle.Render("✗")` → `ui.Error`
- Line 162: success with `ui.SuccessStyle.Render("✓")` → `ui.Success`
- Line 167: error with `ui.ErrorStyle.Render("✗")` → `ui.Error`

- [ ] **Step 1: Add spinner to fetch operation**

In `cmd/install.go`, replace lines 50-55:
```go
// Before:
fmt.Printf("  Fetching from %s...\n", src.Raw)
localPath, sha, err := registry.FetchToCache(src)
if err != nil {
    return fmt.Errorf("fetching source: %w", err)
}
fmt.Printf("  %s\n\n", ui.DimStyle.Render(fmt.Sprintf("commit: %s", sha)))
// After:
var localPath string
var sha string
if spinErr := ui.WithSpinner(fmt.Sprintf("Fetching from %s", src.Raw), func() error {
    var fetchErr error
    localPath, sha, fetchErr = registry.FetchToCache(src)
    return fetchErr
}); spinErr != nil {
    return fmt.Errorf("fetching source: %w", spinErr)
}
fmt.Println(ui.Success(fmt.Sprintf("Fetched %s", ui.DimStyle.Render(sha))))
fmt.Println()
```

- [ ] **Step 2: Migrate error and success messages in install loop**

In `cmd/install.go`, replace lines 125, 134, 158, 162, 167:
```go
// Line 125 — Before:
fmt.Printf("  %s Skipping %s: %v\n", ui.ErrorStyle.Render("✗"), filepath.Base(sp), err)
// After:
fmt.Println(ui.Error(fmt.Sprintf("Skipping %s: %v", filepath.Base(sp), err), ""))

// Line 134 — Before:
fmt.Println(ui.ErrorStyle.Render("  Blocked — use --force to override"))
// After:
fmt.Println(ui.Error("Blocked", "use --force to override"))

// Line 158 — Before:
fmt.Printf("  %s Failed to install to %s: %v\n", ui.ErrorStyle.Render("✗"), a.Config.Name, err)
// After:
fmt.Println(ui.Error(fmt.Sprintf("Failed to install to %s: %v", a.Config.Name, err), ""))

// Line 162 — Before:
fmt.Printf("  %s Installed to %s\n", ui.SuccessStyle.Render("✓"), a.Config.Name)
// After:
fmt.Println(ui.Success(fmt.Sprintf("Installed to %s", a.Config.Name)))

// Line 167 — Before:
fmt.Printf("  %s Failed to record install: %v\n", ui.ErrorStyle.Render("✗"), err)
// After:
fmt.Println(ui.Error(fmt.Sprintf("Failed to record install: %v", err), ""))
```

- [ ] **Step 3: Verify build succeeds**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./...`
Expected: successful build

- [ ] **Step 4: Commit**

```bash
git add cmd/install.go
git commit -m "refactor(install): migrate to ui helpers + add fetch spinner"
```

---

### Task 8: Migrate `sync.go` to use helpers

**Files:**
- Modify: `cmd/sync.go`

The sync command has a lot of per-result output that uses a consistent pattern already (icon + name + agent + status). The main migration targets are:
- Line 85: success message → `ui.Success`
- Line 216: summary line — keep as-is (it's a summary, not a success/error)
- Sync output lines (184-210) — these use a deliberate per-line format that's already consistent internally. Don't change these; they're a progress log, not success/error messages.

- [ ] **Step 1: Migrate success message**

In `cmd/sync.go`, replace line 85:
```go
// Before:
fmt.Printf("%s Saved distribution targets: %s\n\n", ui.SuccessStyle.Render("✓"), strings.Join(selected, ", "))
// After:
fmt.Println(ui.Success(fmt.Sprintf("Saved distribution targets: %s", strings.Join(selected, ", "))))
fmt.Println()
```

- [ ] **Step 2: Verify build succeeds**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./...`
Expected: successful build

- [ ] **Step 3: Commit**

```bash
git add cmd/sync.go
git commit -m "refactor(sync): migrate to ui helper functions"
```

---

### Task 9: Migrate `update_rules.go` to use helpers + add spinner

**Files:**
- Modify: `cmd/update_rules.go`

Current patterns:
- Line 40: "Fetching rules..." → wrap in spinner
- Lines 57, 59, 83: success messages → `ui.Success`
- Line 108: success message → `ui.Success`

The update-rules command has two code paths (pull existing repo, clone fresh). Both are network operations that should get a spinner.

- [ ] **Step 1: Refactor to extract the fetch operation into a helper**

The current function has interleaved I/O and output. Extract the fetch logic into a pure function that returns results, then wrap it with a spinner.

In `cmd/update_rules.go`, restructure `runUpdateRules`:

```go
func runUpdateRules(cmd *cobra.Command, args []string) error {
	coachDir := config.DefaultCoachDir()
	if err := config.EnsureCoachDir(coachDir); err != nil {
		return err
	}

	cfg, err := config.Load(coachDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	rulesDir := filepath.Join(coachDir, "rules")
	repoDir := filepath.Join(rulesDir, "repo")

	var sha string
	var alreadyUpToDate bool

	if spinErr := ui.WithSpinner(fmt.Sprintf("Fetching rules from %s", cfg.RulesSource), func() error {
		var fetchErr error
		sha, alreadyUpToDate, fetchErr = fetchRules(repoDir, cfg.RulesSource)
		return fetchErr
	}); spinErr != nil {
		return fmt.Errorf("fetching rules: %w", spinErr)
	}

	if alreadyUpToDate {
		fmt.Println(ui.Success("Already up to date."))
	} else {
		fmt.Println(ui.Success(fmt.Sprintf("Updated to %s", sha)))
	}

	return copyRuleFiles(repoDir, rulesDir)
}

func fetchRules(repoDir, source string) (sha string, upToDate bool, err error) {
	if _, statErr := os.Stat(filepath.Join(repoDir, ".git")); statErr == nil {
		repo, openErr := git.PlainOpen(repoDir)
		if openErr == nil {
			w, wtErr := repo.Worktree()
			if wtErr == nil {
				pullErr := w.Pull(&git.PullOptions{Force: true})
				if pullErr != nil && pullErr != git.NoErrAlreadyUpToDate {
					os.RemoveAll(repoDir)
				} else {
					head, _ := repo.Head()
					sha = "unknown"
					if head != nil {
						sha = head.Hash().String()[:12]
					}
					return sha, pullErr == git.NoErrAlreadyUpToDate, nil
				}
			}
		}
	}

	os.RemoveAll(repoDir)
	repo, cloneErr := git.PlainClone(repoDir, false, &git.CloneOptions{
		URL:           source,
		Depth:         1,
		ReferenceName: plumbing.HEAD,
		SingleBranch:  true,
	})
	if cloneErr != nil {
		return "", false, fmt.Errorf("cloning rules repository: %w\n\n  If the rules repo doesn't exist yet, this is expected.\n  Coach will use its embedded patterns in the meantime.", cloneErr)
	}

	head, _ := repo.Head()
	sha = "unknown"
	if head != nil {
		sha = head.Hash().String()[:12]
	}
	return sha, false, nil
}
```

- [ ] **Step 2: Migrate `copyRuleFiles` success message**

In `cmd/update_rules.go`, replace the success line in `copyRuleFiles`:
```go
// Before:
fmt.Printf("  %s Updated %d rule files\n", ui.SuccessStyle.Render("✓"), copied)
// After:
fmt.Println(ui.Success(fmt.Sprintf("Updated %d rule files", copied)))
```

- [ ] **Step 3: Verify build succeeds**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./...`
Expected: successful build

- [ ] **Step 4: Commit**

```bash
git add cmd/update_rules.go
git commit -m "refactor(update-rules): migrate to ui helpers + add fetch spinner"
```

---

### Task 10: Migrate `scan.go` to use helpers

**Files:**
- Modify: `cmd/scan.go`

Current patterns:
- Line 74: stderr error → `ui.Error`
- Lines 128-134: risk-based messages — these are already well-formatted using styles directly, but should use helpers for consistency
- Line 68: "No managed skills found." — plain text, should use `ui.Warn`

- [ ] **Step 1: Migrate output patterns**

In `cmd/scan.go`:

```go
// Line 68 — Before:
fmt.Println("No managed skills found.")
// After:
fmt.Println(ui.Warn("No managed skills found."))

// Line 74 — Before:
fmt.Fprintf(os.Stderr, "  %s  %s: %v\n", ui.ErrorStyle.Render("✗"), m.Name, err)
// After:
fmt.Fprintln(os.Stderr, ui.Error(fmt.Sprintf("%s: %v", m.Name, err), ""))

// Lines 126-136 — Before:
switch result.Risk {
case pkg.RiskLow:
    fmt.Println(ui.SuccessStyle.Render("  Safe to install."))
case pkg.RiskMedium:
    fmt.Println(ui.WarningStyle.Render("  Review warnings before installing."))
case pkg.RiskHigh:
    fmt.Println(ui.ErrorStyle.Render("  Manual review recommended before installing."))
case pkg.RiskCritical:
    fmt.Println(ui.ErrorStyle.Render("  DO NOT install without thorough review."))
    os.Exit(1)
}
// After:
switch result.Risk {
case pkg.RiskLow:
    fmt.Println(ui.Success("Safe to install."))
case pkg.RiskMedium:
    fmt.Println(ui.Warn("Review warnings before installing."))
case pkg.RiskHigh:
    fmt.Println(ui.Error("Manual review recommended before installing.", ""))
case pkg.RiskCritical:
    fmt.Println(ui.Error("DO NOT install without thorough review.", ""))
    os.Exit(1)
}
```

- [ ] **Step 2: Verify build succeeds**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./...`
Expected: successful build

- [ ] **Step 3: Commit**

```bash
git add cmd/scan.go
git commit -m "refactor(scan): migrate to ui helper functions"
```

---

### Task 11: Migrate `lint.go` and `status.go` to use helpers

**Files:**
- Modify: `cmd/lint.go`
- Modify: `cmd/status.go`

- [ ] **Step 1: Migrate lint.go**

In `cmd/lint.go`:
```go
// Line 72 — Before:
fmt.Println("No managed skills found.")
// After:
fmt.Println(ui.Warn("No managed skills found."))
```

The rest of lint's output uses `RenderScanSummary` and `RenderFindings` which are fine as-is.

- [ ] **Step 2: Migrate status.go**

In `cmd/status.go`:
```go
// Lines 33-34 — Before:
fmt.Println(ui.WarningStyle.Render("  No coding agents detected."))
fmt.Println(ui.DimStyle.Render("  Coach looks for Claude Code, Cursor, Codex, and Copilot."))
// After:
fmt.Println(ui.Warn("No coding agents detected."))
fmt.Println(ui.DimStyle.Render("  Coach looks for Claude Code, Cursor, Codex, and Copilot."))
```

- [ ] **Step 3: Verify build and all tests pass**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build ./... && go test ./...`
Expected: successful build, all tests pass

- [ ] **Step 4: Commit**

```bash
git add cmd/lint.go cmd/status.go
git commit -m "refactor(lint,status): migrate to ui helper functions"
```

---

### Task 12: Final verification

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go test ./... -v`
Expected: all tests pass

- [ ] **Step 2: Build binary and smoke test**

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && go build -o coach . && ./coach --help && ./coach status`
Expected: clean help output, status works

- [ ] **Step 3: Verify no remaining ad-hoc patterns in migrated commands**

Search for old patterns that should have been replaced:
Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && grep -n 'SuccessStyle.Render("✓")' cmd/edit.go cmd/generate.go cmd/init.go cmd/install.go cmd/update_rules.go cmd/scan.go cmd/status.go`
Expected: no matches (all migrated to `ui.Success`)

Run: `cd /Users/dylan/.superset/worktrees/coach/feat/better-ux-flows && grep -n 'ErrorStyle.Render.*✗' cmd/edit.go cmd/generate.go cmd/install.go cmd/scan.go`
Expected: no matches (all migrated to `ui.Error`)

- [ ] **Step 4: Commit any final cleanup if needed**

Only if the grep search reveals missed patterns.
