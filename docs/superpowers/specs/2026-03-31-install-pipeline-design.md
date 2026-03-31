# Install Pipeline Design

**Date:** 2026-03-31
**Status:** Draft

## Summary

Replace the current single-skill `coach install` command with a batch-aware pipeline that discovers, evaluates, and presents skills for interactive approval before committing them to coach's managed store. Supports local directories, remote GitHub repos, and auditing already-installed agent skills.

## Motivation

Skills accumulate across local directories, agent installations, and remote repos without consistent vetting. The current `coach install` processes one skill at a time with only a security scan gate. There's no batch discovery, no quality checks, no way to audit skills that were manually placed in agent directories, and no interactive review step.

## Command Interface

```bash
coach install <source>                 # batch discover + pipeline + interactive selection
coach install <source> --yes           # auto-approve all passing skills (skip TUI)
coach install <source> --agent cursor  # install to specific agent only
coach install <source> --scope global  # override default_scope config
coach install <source> --copy          # copy files instead of symlinking
coach install <source> --force         # allow selecting CRITICAL-flagged skills
coach install --installed              # audit untracked/modified skills in agent dirs
```

Source types (unchanged from existing `ParseSource`):
- Local paths: `./path`, `/absolute/path`
- GitHub shorthand: `owner/repo`
- GitHub URLs: `https://github.com/owner/repo`

## Architecture

Four-stage pipeline orchestrated from `cmd/install.go`:

```
Source(s) → Discover → Evaluate → Present → Commit
               │           │          │         │
         []SkillCandidate  │    user selection   │
                    []VettedSkill          []InstalledSkill
```

New package `internal/pipeline` exposes three functions. The TUI (progress bar + selection table) lives in `internal/ui/`.

The `--skill <name>` flag filters Discover output before passing to Evaluate, preserving existing behavior for targeting a specific skill from a multi-skill source.

### Stage 1: Discover

`Discover(source Source, installed bool, agents []DetectedAgent, provenance *InstalledSkills) ([]SkillCandidate, error)`

When `installed` is true, ignores `source` and audits agent skill directories instead. Otherwise, discovers from the given source. Returns a flat list of candidates. Three discovery modes:

**Local path:**
- Resolve to absolute path
- Walk directory finding SKILL.md files (reuses `registry.FindSkills`)
- Flat layout (SKILL.md at root) → single candidate
- Nested layout (subdirs with SKILL.md) → one candidate per subdir

**Remote source:**
- `ParseSource` → `FetchToCache` → `FindSkills` on cached clone
- Same flat/nested detection as local
- Commit SHA captured for provenance

**Installed audit (`--installed`):**
- Iterate detected agents' skill directories
- Find SKILL.md files not in coach's provenance (`installed.yaml`) → `InstalledUntracked`
- Find SKILL.md files that match a provenance name but have different content hash → `InstalledModified`
- Skip skills where name and content hash both match provenance (already vetted, unchanged)

```go
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
```

### Stage 2: Evaluate

`Evaluate(candidates []SkillCandidate, db *rules.PatternDB, onProgress func(current, total int, name string)) ([]VettedSkill, error)`

Runs three checks per candidate. The `onProgress` callback drives the progress bar TUI.

**Check 1: Lint (spec compliance)**
- Reuses `skill.Parse` + `skill.Validate`
- Catches: missing fields, bad name format, oversized fields, empty body
- Result: PASS or FAIL with list of violations

**Check 2: Scan (security)**
- Reuses `scanner.ScanSkill` with loaded pattern DB
- Catches: prompt injection, dangerous commands, suspicious scripts
- Result: risk level (NONE/LOW/MEDIUM/HIGH/CRITICAL) + findings

**Check 3: Quality (new)**
- New function `skill.CheckQuality(s *types.Skill) CheckResult`
- Lives in `internal/skill/quality.go`
- Checks:
  - Description lacks "Use when" / "Use this" trigger phrase → WARN
  - No `allowed-tools` declared in frontmatter → WARN
  - No "When to Use" / "## When" section in body → WARN
  - Body under 50 chars (suspiciously thin) → WARN
  - Description under 20 chars (vague) → WARN
- Quality issues are warnings only — never block selection

```go
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
    Skill         *types.Skill      // parsed skill, nil if lint failed
    LintResult    CheckResult
    ScanResult    *types.ScanResult
    QualityResult CheckResult
    Selectable    bool              // false if lint FAIL or scan CRITICAL (without --force)
}
```

### Stage 3: Present (Interactive Selection)

Bubble Tea model with two phases.

**Phase 1: Progress bar**

Uses `bubbles/progress`. Displays during the Evaluate stage:

```
 Evaluating skills from /vault/skills/

 ████████████████░░░░░░░░░░  14/18

 Scanning: prd-to-issues
```

Transitions to the selection table when evaluation completes.

**Phase 2: Selection table**

Built with `bubbles/table` + custom rendering for selection state and status indicators:

```
 Coach Install — 18 skills discovered from /vault/skills/

 Sel │ Skill                     │ Lint │ Scan │ Quality │ Issues
─────┼───────────────────────────┼──────┼──────┼─────────┼─────────────────────
 [ ] │ triage-issue              │ PASS │ PASS │ PASS    │
 [ ] │ grill-me                  │ PASS │ PASS │ WARN    │ no allowed-tools
 [ ] │ write-a-skill             │ PASS │ PASS │ PASS    │
 [!] │ sketchy-plugin            │ PASS │ CRIT │ —       │ prompt injection
 [✗] │ scaffold-exercises        │ FAIL │ —    │ —       │ missing description
 [~] │ my-modified-skill         │ PASS │ PASS │ PASS    │ modified since install

 0 selected

 [a] all passing  [n] none  [↑↓] navigate  [space] toggle  [p] preview  [enter] confirm  [q] quit
```

**Row states:**
- `[ ]` — selectable, unselected (default for all selectable skills)
- `[●]` — selectable, selected
- `[!]` — CRITICAL scan, unselectable (unless `--force`)
- `[✗]` — lint failure, unselectable
- `[~]` — modified since install (from `--installed`), selectable

**Hotkeys:**
- `a` — select all passing (all selectable skills toggled on)
- `n` — select none (reset to default)
- `↑`/`↓` or `j`/`k` — navigate rows
- `space` — toggle current row
- `p` — inline preview of highlighted skill (renders SKILL.md body below table, `p` or `esc` to dismiss)
- `enter` — confirm selection, proceed to scope prompt
- `q` — quit, install nothing

**After confirm**, a `huh` form asks for scope:

```
Install 7 skills to: (global) / (local) / (cancel)
```

Pre-selects based on `default_scope` from config or `--scope` flag.

Final confirmation:

```
Install 7 skills to ~/.coach/skills/ (global). Proceed? [y/N]
```

**`--yes` mode:** Skips the entire TUI. Auto-selects all skills where lint passed and scan is below CRITICAL. Uses `--scope` flag or config default. No confirmation prompt.

### Stage 4: Commit

`Commit(selected []VettedSkill, agents []DetectedAgent, scope string, opts InstallOptions) ([]InstalledSkill, error)`

For each selected skill:

1. Copy or symlink to scope directory (`~/.coach/skills/` for global, `.coach/skills/` for local)
2. Compute SHA-256 content hash of SKILL.md for future `--installed` audits
3. Distribute to configured agents via `registry.InstallSkill`
4. Record provenance in `~/.coach/installed.yaml`

Provenance record (extended from existing):

```go
type InstalledSkill struct {
    Name        string
    Source      string
    CommitSHA   string    // git SHA for remote, "local" for local
    ContentHash string    // SHA-256 of SKILL.md at install time
    InstallDate time.Time
    RiskScore   int
    Agents      []string
}
```

Post-install summary printed to terminal:

```
 Installed 7 skills (global)

  ✓ triage-issue        → claude-code, cursor
  ✓ grill-me            → claude-code, cursor
  ✓ write-a-skill       → claude-code, cursor
  ...

 Run 'coach list' to see all installed skills.
```

## File Layout

New and modified files:

```
internal/
  pipeline/
    discover.go       # Discover function + SkillCandidate type
    evaluate.go       # Evaluate function + VettedSkill type
    commit.go         # Commit function + post-install summary
    pipeline.go       # shared types (OriginType, CheckResult, CheckStatus)
  skill/
    quality.go        # CheckQuality function (new)
  ui/
    install_model.go  # Bubble Tea model (progress bar + selection table)
  types/
    types.go          # ContentHash field added to InstalledSkill
  registry/
    install.go        # ContentHash computation added to RecordInstall
cmd/
  install.go          # Rewritten to orchestrate pipeline stages
```

## Backwards Compatibility

- `coach install <source>` with a single-skill source shows a one-row table (or use `--yes` for old behavior)
- `--skill <name>` flag preserved — filters discovery results before evaluation
- `--list` flag preserved — runs Discover only, prints available skills, exits
- `--copy`, `--force`, `--agent` flags unchanged in semantics
- New flags: `--scope`, `--installed`, `--yes`
- Provenance records gain `ContentHash` field; existing records without it are treated as "no hash available" (always shown in `--installed` audit)
