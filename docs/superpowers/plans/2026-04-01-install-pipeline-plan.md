# Install Pipeline — Implementation Plan

**Spec:** `docs/superpowers/specs/2026-03-31-install-pipeline-design.md`
**Issue:** #24
**Branch:** `feat/batch-install`

## Overview

Seven steps, each independently testable. Steps 1–4 build the pipeline package bottom-up. Step 5 adds the TUI. Step 6 rewires `cmd/install.go`. Step 7 adds the quality checker.

---

## Step 1: Shared types + ContentHash in provenance

**What:** Add pipeline types to a new `internal/pipeline/pipeline.go` and extend `InstalledSkill` with a `ContentHash` field.

**Files:**

| File | Action |
|------|--------|
| `internal/pipeline/pipeline.go` | **New** — `OriginType`, `SkillCandidate`, `CheckStatus`, `CheckResult`, `VettedSkill`, `InstallOptions` types |
| `internal/types/types.go` | **Edit** — add `ContentHash string` field to `InstalledSkill` |

**Types (pipeline.go):**

```go
package pipeline

type OriginType int
const (
    OriginLocal OriginType = iota
    OriginRemote
    OriginInstalledUntracked
    OriginInstalledModified
)

type SkillCandidate struct {
    Path   string     // absolute path to skill directory
    Source string     // original source string for provenance
    SHA    string     // commit SHA if remote, "local" otherwise
    Origin OriginType
}

type CheckStatus int
const (
    CheckPass CheckStatus = iota
    CheckWarn
    CheckFail
)

type CheckResult struct {
    Status CheckStatus
    Issues []string
}

type VettedSkill struct {
    Candidate     SkillCandidate
    Skill         *types.Skill
    LintResult    CheckResult
    ScanResult    *types.ScanResult
    QualityResult CheckResult
    Selectable    bool
}

type InstallOptions struct {
    Copy   bool
    Force  bool
    Scope  string // "global" or "local"
    Agents []types.DetectedAgent
}
```

**InstalledSkill change:** Add `ContentHash string \`yaml:"content_hash,omitempty"\`` after `RiskScore`. Existing records without it unmarshal as empty string (backwards compatible).

**Tests:** Confirm YAML round-trip of `InstalledSkill` with and without `ContentHash`.

**Verification:** `go build ./...` passes. Existing tests pass.

---

## Step 2: Discover

**What:** Implement the three discovery modes: local path, remote source, installed audit.

**Files:**

| File | Action |
|------|--------|
| `internal/pipeline/discover.go` | **New** — `Discover` function |
| `internal/pipeline/discover_test.go` | **New** — unit tests |

**`Discover` signature:**

```go
func Discover(src *registry.Source, installed bool, agents []types.DetectedAgent, provenance *registry.InstalledSkills) ([]SkillCandidate, error)
```

**Logic by mode:**

- **`installed == true`:** Iterate each agent's `SkillDir`, find SKILL.md files. For each, check if name exists in provenance. If not → `OriginInstalledUntracked`. If yes, compute SHA-256 of SKILL.md content and compare to `ContentHash` in provenance → `OriginInstalledModified` if different, skip if same.
- **Local (`src.Type == SourceLocal`):** Resolve to absolute path. Call `registry.FindSkills`. Map each to `SkillCandidate` with `Origin: OriginLocal`, `SHA: "local"`.
- **Remote:** Call `registry.FetchToCache(src)`. Then `registry.FindSkills` on the cached path. Map each to `SkillCandidate` with `Origin: OriginRemote`, `SHA` from fetch.

**Helper:** `func contentHash(path string) (string, error)` — reads `SKILL.md`, returns hex-encoded SHA-256.

**Tests:**
- Local dir with 3 skills → 3 candidates
- Flat layout (single SKILL.md at root) → 1 candidate
- Installed audit: skill in agent dir not in provenance → `OriginInstalledUntracked`
- Installed audit: skill with matching hash → skipped
- Installed audit: skill with different hash → `OriginInstalledModified`

---

## Step 3: Evaluate

**What:** Run lint + scan + quality on each candidate. Report progress via callback.

**Files:**

| File | Action |
|------|--------|
| `internal/pipeline/evaluate.go` | **New** — `Evaluate` function |
| `internal/pipeline/evaluate_test.go` | **New** — unit tests |

**`Evaluate` signature:**

```go
func Evaluate(candidates []SkillCandidate, db *types.PatternDatabase, force bool, onProgress func(current, total int, name string)) ([]VettedSkill, error)
```

**Per-candidate logic:**

1. **Lint:** `skill.Parse(candidate.Path)` — if error, `LintResult = CheckFail` with error message, `Skill = nil`, `Selectable = false`. If success, run `skill.Validate` — any errors → `CheckFail`, otherwise `CheckPass`.
2. **Scan:** Only if lint passed. `scanner.ScanSkill(skill, db)`. Store full `ScanResult`. If `Risk == RiskCritical` and `!force` → `Selectable = false`.
3. **Quality:** Only if lint passed. Call `skill.CheckQuality(skill)` (new function, Step 7). Always `CheckWarn` or `CheckPass`, never blocks.
4. Call `onProgress(i+1, len(candidates), skillName)`.

**Selectable rule:** `true` unless lint failed OR (scan CRITICAL AND !force).

**Tests:**
- Valid skill → all three checks run, `Selectable = true`
- Invalid SKILL.md (missing name) → `LintResult.Status == CheckFail`, scan/quality skipped
- Skill with CRITICAL scan result → `Selectable = false` (without force), `true` (with force)

---

## Step 4: Commit

**What:** Install selected skills to the chosen scope directory, compute content hash, record provenance, distribute to agents.

**Files:**

| File | Action |
|------|--------|
| `internal/pipeline/commit.go` | **New** — `Commit` function |
| `internal/pipeline/commit_test.go` | **New** — unit tests |

**`Commit` signature:**

```go
type CommitResult struct {
    Name   string
    Agents []string
    Err    error
}

func Commit(selected []VettedSkill, coachDir string, opts InstallOptions) ([]CommitResult, error)
```

**Per-skill logic:**

1. Determine target dir: if `opts.Scope == "global"` → `coachDir/skills/<name>`, if `"local"` → `.coach/skills/<name>` (relative to cwd).
2. Copy or symlink source to target dir via `registry.InstallSkill`.
3. Compute content hash of the installed SKILL.md.
4. For each agent in `opts.Agents`, call `registry.InstallSkill(targetDir, agent.SkillDir, ...)` to distribute.
5. Call `registry.RecordInstall` with the content hash included (requires extending `RecordInstall` to accept `ContentHash`).
6. Collect results.

**Changes to existing code:**
- `registry.RecordInstall` — add `contentHash` parameter, set it on the `InstalledSkill` struct.
- `registry.install.go` — update `RecordInstall` signature.

**Tests:**
- Install 2 skills to temp dir → both exist in scope dir + agent dirs
- Provenance file contains content hashes
- Copy mode creates independent files (not symlinks)

---

## Step 5: Install TUI (Bubble Tea model)

**What:** Two-phase Bubble Tea model: progress bar during evaluation, then interactive selection table.

**Files:**

| File | Action |
|------|--------|
| `internal/ui/install_model.go` | **New** — `InstallModel` Bubble Tea model |
| `internal/ui/install_model_test.go` | **New** — unit tests for state transitions |

**Phase 1: Progress bar**

- Uses `bubbles/progress` component
- Receives progress updates from `Evaluate`'s callback via a channel or tea.Cmd
- Shows: bar, `current/total`, current skill name
- Transitions to Phase 2 when evaluation completes

**Model structure:**

```go
type InstallModel struct {
    // Config
    source    string
    force     bool

    // Phase 1: progress
    phase     int // 0=progress, 1=selection
    progress  progress.Model
    current   int
    total     int
    evalName  string
    vetted    []pipeline.VettedSkill

    // Phase 2: selection
    cursor    int
    selected  map[int]bool
    confirmed bool
    cancelled bool

    // Preview
    previewing bool
    previewIdx int
}
```

**Phase 2: Selection table**

Renders the table from the spec. Row states:
- `[ ]` unselected selectable, `[●]` selected, `[!]` CRITICAL, `[✗]` lint fail, `[~]` modified

**Hotkeys:** `a` (all passing), `n` (none), `↑`/`↓`/`j`/`k` (navigate), `space` (toggle), `p` (preview), `enter` (confirm), `q` (quit)

**Preview:** When `p` pressed, render SKILL.md body below the table. `p` or `esc` dismisses.

**Return value:** The model exposes `Selected() []pipeline.VettedSkill` and `Cancelled() bool` after the program exits.

**Tests:**
- Init with 3 vetted skills → renders table with 3 rows
- `a` key → all selectable skills selected
- `space` on unselectable row → no change
- `q` → cancelled
- `enter` with selections → confirmed, `Selected()` returns chosen skills

---

## Step 6: Rewrite `cmd/install.go`

**What:** Replace the current sequential install logic with the pipeline orchestration.

**Files:**

| File | Action |
|------|--------|
| `cmd/install.go` | **Rewrite** — orchestrate pipeline stages |

**New flags:**
- `--scope` (string, default from config) — `"global"` or `"local"`
- `--installed` (bool) — audit mode
- `--yes` (bool) — auto-approve, skip TUI

**Flow:**

```
1. Parse flags, load config
2. If --installed: call Discover(nil, true, agents, provenance)
   Else: ParseSource → Discover(src, false, agents, provenance)
3. If --skill: filter candidates by name
4. If --list: print candidates and exit
5. Load pattern DB
6. If --yes:
     Evaluate(candidates, db, force, printProgress)
     Auto-select all where Selectable == true
     Prompt for scope (or use --scope)
     Commit
   Else:
     Run InstallModel TUI (handles both evaluate progress + selection)
     If cancelled, exit
     Prompt for scope via huh
     Confirm via huh
     Commit
7. Print summary
```

**Args change:** `cobra.ExactArgs(1)` → `cobra.MaximumNArgs(1)` (0 args allowed when `--installed`).

**Backwards compatibility:**
- Single-skill source → one-row table (or `--yes` for old behavior)
- All existing flags preserved with same semantics

---

## Step 7: Quality checker in `internal/skill/`

**What:** New `CheckQuality` function separate from scanner's quality checks. The scanner's `CheckQuality` handles security-adjacent quality (missing allowed-tools, short description). The pipeline's quality checker adds install-time heuristics.

**Files:**

| File | Action |
|------|--------|
| `internal/skill/quality.go` | **New** — `CheckQuality` function |
| `internal/skill/quality_test.go` | **New** — unit tests |

**Checks:**
- Description lacks "Use when" / "Use this" trigger phrase → WARN
- No "When to Use" / "## When" section in body → WARN
- Body under 50 chars → WARN
- No `allowed-tools` declared → WARN (overlaps with scanner, but pipeline uses this independently)
- Description under 20 chars → WARN

**Signature:**

```go
func CheckQuality(s *types.Skill) pipeline.CheckResult
```

Returns `CheckPass` if no warnings, `CheckWarn` with issues list otherwise.

**Tests:**
- Skill with good description + trigger phrase + allowed-tools → `CheckPass`
- Skill with 10-char description → `CheckWarn` with "description under 20 chars"
- Skill with no "When to Use" section → `CheckWarn`

---

## Implementation Order

```
Step 1 (types)
  ↓
Step 2 (discover) ←── Step 7 (quality) can be parallel
  ↓
Step 3 (evaluate) ←── depends on Step 7
  ↓
Step 4 (commit)
  ↓
Step 5 (TUI)
  ↓
Step 6 (cmd/install.go rewrite)
```

Steps 1 → 2 → 3 → 4 are strictly sequential. Step 7 can be done alongside Step 2 since Step 3 needs it. Step 5 depends on Step 3's types. Step 6 ties everything together.

## Verification

After each step: `go build ./...` and `go test ./...` must pass. After Step 6: manual testing with:
- `coach install ./testdata` (local multi-skill)
- `coach install owner/repo` (remote)
- `coach install --installed` (audit)
- `coach install ./testdata --yes` (non-interactive)
- `coach install ./testdata --list` (list only)
