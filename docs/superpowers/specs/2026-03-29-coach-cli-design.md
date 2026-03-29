# Coach CLI — Design Spec

**Date:** 2026-03-29
**Status:** Draft

## Overview

Coach is a Go CLI tool for developing, testing, and managing AI agent skills and configurations. It is the developer experience and governance layer for the Agent Skills ecosystem — not a package manager, but the authoring, testing, security, and team management tool that sits alongside existing distribution channels (skills.sh, ClawHub, GitHub repos).

### Positioning

The Agent Skills spec (agentskills.io) standardizes the file format. Distribution tools like `npx skills` handle installation from GitHub. Coach fills the gap that neither addresses: how do you write a good skill, know it works, know it's safe, and manage skills across your agents and your team?

### Business Model

- Free for individual developers (all authoring, testing, scanning, and management commands)
- Paid when teams are involved (shared registries, trust policies, cross-team audit)
- Bottom-up GTM: the solo experience must be indispensable so developers push for team adoption

## Technical Architecture

### Language & Distribution

Written in Go. Distributed as a single static binary — no runtime dependencies, no package managers required. Cross-compiled for macOS (arm64, amd64), Linux (arm64, amd64), and Windows.

### Project Structure

```
coach/
  cmd/              # CLI command definitions (Cobra for arg parsing)
  internal/
    skill/          # SKILL.md parsing, validation, spec compliance
    scanner/        # Security analysis engine
    tester/         # Scenario-based skill testing
    registry/       # Fetching/installing from GitHub, local sources
    agent/          # Agent detection and config management
    config/         # Coach's own config (~/.coach/)
    rules/          # Embedded + updatable security patterns and agent registry
    ui/             # Bubbletea TUI components, Lipgloss styles
  pkg/              # Shared types and interfaces
```

### Dependencies

| Need | Choice | Reason |
|---|---|---|
| CLI arg parsing | Cobra | Standard Go CLI framework, handles subcommands/flags |
| TUI framework | Bubbletea | Interactive flows for init wizards, test progress, dashboards |
| Forms | Huh | Built on Bubbletea, great for `coach init` scaffolding prompts |
| Terminal styling | Lipgloss | Consistent styled output across all commands |
| Markdown rendering | Glamour | Preview SKILL.md in terminal |
| Pre-built TUI components | Bubbles | Spinners, tables, text inputs, file pickers |
| Logging | Charm Log | Structured, styled logging |
| YAML parsing | go-yaml | SKILL.md frontmatter parsing |
| Git operations | go-git | Clone/fetch without shelling out to git |
| Markdown AST | Goldmark | Structural validation of skill bodies |

### Coach Config Directory

```
~/.coach/
  config.yaml           # Global settings (default agents, trusted sources, scan preferences)
  trust/                # Signed trust records for vetted skills
  cache/                # Downloaded skill repos, scan results
  rules/                # Remote-updated security patterns and agent registry
  installed.yaml        # Provenance records for all installed skills
  team/                 # Team config (synced when on a team plan)
```

## Agent Detection

Coach auto-detects installed agents by checking for known config directories. The agent registry is stored as embedded YAML data, updatable via `coach update-rules` without shipping a new binary.

### Agent Registry Format

```yaml
agents:
  claude-code:
    skill_dir: "~/.claude/skills/"
    project_skill_dir: ".claude/skills/"
    config_files: ["CLAUDE.md"]
    supports:
      skills: true
      hooks: true
      mcp: true
  cursor:
    skill_dir: "~/.cursor/rules/"
    config_files: [".cursorrules", ".cursor/rules/*.mdc"]
    supports:
      skills: true
      hooks: false
      mcp: true
  codex:
    skill_dir: "~/.codex/skills/"
    project_skill_dir: ".agents/skills/"
    config_files: ["AGENTS.md"]
    supports:
      skills: true
      hooks: false
      mcp: false
```

Detection is a directory existence check per agent. When a command needs agent context (install, status, doctor), it runs the sweep and presents results.

## Commands

### v0.1 — "Daily Driver"

#### `coach init skill`

Scaffolds a new skill with an interactive Huh form.

**Flow:**
1. Prompt for skill name, description, license
2. Ask which features to include (scripts/, references/, tests/)
3. Ask about tool restrictions (allowed-tools)
4. Generate SKILL.md with valid frontmatter and skeleton content
5. Create directory structure

**Output:** A ready-to-edit skill directory that passes `coach lint` out of the box.

#### `coach lint [path]`

Static analysis — spec compliance and security in one pass.

**Checks:**
- Frontmatter field validation (required fields, lengths, allowed values per agentskills.io spec)
- Markdown structure quality (description clarity, instruction coherence)
- Prompt injection pattern scanning (known attack vectors)
- Script analysis (dangerous shell patterns, credential access, network exfiltration)
- `allowed-tools` audit (does the skill request more permissions than it needs?)
- File structure validation (unexpected files, oversized assets)

**Output:** Styled results with severity levels (error/warning/info). Exit code 1 on errors, 0 on clean or warnings-only.

**Flags:**
- `--json` — machine-readable output for CI integration

#### `coach scan <path|source>`

Deep security analysis. Can run on local paths or remote sources (fetches before scanning).

**Detection categories:**

*Prompt Injection (highest severity):*
- Instruction override patterns ("ignore previous instructions", "you are now", "system prompt:")
- Hidden instructions via unicode (zero-width characters, RTL overrides, homoglyph substitution)
- Base64/hex encoded payloads in markdown
- Markdown comment blocks containing hidden instructions
- References to external URLs that could serve dynamic payloads

*Script Analysis:*
- Dangerous shell patterns (curl piping to shell, eval, exec, obfuscated code)
- Credential access (~/.ssh, ~/.aws, environment variable harvesting)
- Network exfiltration (outbound HTTP to unexpected destinations)
- File system operations outside expected scope

*Quality Warnings:*
- Overly broad descriptions (likely to activate incorrectly)
- Missing allowed-tools (no declared permission boundaries)
- Conflicting instructions within the skill body
- Excessive length

**Risk Scoring:**

Each skill gets a score from 0-100:
- 0-25: LOW — safe to install
- 26-50: MEDIUM — review warnings before installing
- 51-75: HIGH — manual review recommended
- 76-100: CRITICAL — blocked from install by default (--force to override)

#### `coach install <source>`

Install skills with automatic security scanning.

**Sources:** GitHub shorthand (`owner/repo`), full URLs, local file paths.

**Flow:**
1. Fetch from source (clone/pull via go-git)
2. Run `coach lint` + `coach scan` automatically
3. If warnings found, prompt user with styled summary and ask to proceed
4. If critical issues found, block unless `--force`
5. Auto-detect installed agents
6. Symlink skill into each agent's skill directory (or `--agent <name>` to target one, `--copy` for independent copies)
7. Record provenance in `~/.coach/installed.yaml` (source URL, commit SHA, install date, scan result, risk score)

**Flags:**
- `--agent <name>` — install to specific agent only
- `--copy` — copy instead of symlink
- `--force` — override critical security blocks
- `--list` — list available skills in source before installing
- `--skill <name>` — install specific skill from a multi-skill repo

#### `coach status`

Dashboard view of agent setup. Rendered as a styled Lipgloss table.

**Shows:**
- All detected agents and their config locations
- Installed skills per agent with source, commit SHA, and last-scanned status
- Flags outdated skills (source repo has newer commits)
- Flags unvetted skills (installed outside Coach, no provenance record)
- Summary counts (total, unvetted, outdated)

#### `coach preview <path>`

Renders a SKILL.md in the terminal using Glamour. Shows parsed frontmatter fields, rendered markdown body, and file tree of the skill directory. Exactly what an agent would see when the skill activates.

#### `coach update-rules`

Fetches latest security patterns and agent registry from a remote source.

**Mechanics:**
- Source: a GitHub repo maintained by the Coach project (e.g., `coach-dev/security-rules`)
- Downloads to `~/.coach/rules/`
- Remote rules merge with and take priority over the embedded baseline
- No auto-fetch — explicitly user-triggered, keeps the tool predictable and offline-friendly
- Shows diff summary of what changed (new patterns added, agents updated)

### v0.2 — "Confidence"

#### `coach test [path]`

Scenario-based skill testing with a tiered approach.

**Test file format:** YAML files in a `tests/` directory alongside the skill.

```yaml
name: "Activates on React component questions"
trigger: "Build me a login form component in React"
expect:
  activates: true
  contains:
    - "accessibility"
    - "component structure"
  tools_called:
    - Write
  no_contains:
    - "jQuery"
```

**Tier 1 — Static (no LLM, instant):**
- Does the trigger match the skill's description? (keyword/pattern matching)
- Does the SKILL.md parse correctly?
- Are referenced files present and valid?
- Runs by default on every `coach test`

**Tier 2 — Simulation (local, fast):**
- Simulates agent skill selection: given a trigger and installed skills, would this one activate?
- Tests for conflicts: would multiple skills activate on the same prompt?
- Uses embedding similarity (lightweight local model)
- Run with `coach test --sim`

**Tier 3 — LLM Evaluation (requires API key):**
- Sends trigger to a model with the skill loaded
- Validates response matches expect criteria
- Supports multi-turn scenarios
- Run with `coach test --eval`
- Configurable model via `--model`

**Output:** Styled test runner output (pass/fail/skip with timing, diffs on failure).

**Key constraint:** Tier 1 and 2 must be useful without API keys. The free experience cannot depend on paid inference.

#### `coach doctor`

Diagnoses problems with agent setup.

**Checks:**
- Broken symlinks in agent skill directories
- Conflicting skills (overlapping triggers/descriptions)
- Agent config directory structure validation
- Spec violations across installed skills
- Coach config integrity

**Output:** Checklist of issues with suggested fix commands.

#### `coach init hook`

Scaffolds a hook configuration for a specific agent lifecycle event. Interactive form to select agent, event type, and action.

#### `coach init agents-md`

Generates an AGENTS.md by inspecting the current project — reads package.json, go.mod, Makefile, Cargo.toml, etc. to pre-fill build commands, test commands, and project structure description.

#### `coach lint --fix`

Auto-fixes common lint issues: missing frontmatter fields, formatting problems, spec compliance gaps.

### v0.3 — "Teams"

#### `coach team` namespace

- `coach team init` — set up a team registry (backed by a Git repo)
- `coach team publish <path>` — publish a vetted skill to the team registry
- `coach team pull` — sync team-approved skills to local setup
- `coach team audit` — view installed skills across team members
- `coach team policy` — manage trust policies and allowlists

Team registries are Git repos with a defined structure. No hosted infrastructure required in v0.3 — that's a later product decision.

#### `coach test --eval`

Tier 3 LLM-based evaluation. Requires an API key configured in `~/.coach/config.yaml`.

## Security Scanning Engine

### Pattern Database

Security patterns ship as an embedded ruleset compiled into the binary. This embedded set is the baseline — always available offline.

Remote updates via `coach update-rules` download additional patterns to `~/.coach/rules/`. At runtime, remote rules merge with embedded rules. Remote rules take priority (they're newer).

### Pattern Format

```yaml
patterns:
  - id: "PI-001"
    category: "prompt-injection"
    severity: "critical"
    name: "Instruction override"
    description: "Attempts to override agent instructions"
    regex: "(?i)(ignore|disregard|forget)\\s+(all\\s+)?(previous|prior|above)\\s+(instructions|prompts|rules)"
    file_types: ["*.md"]
  - id: "SC-001"
    category: "script-danger"
    severity: "high"
    name: "Pipe to shell"
    description: "Downloads and executes remote code"
    regex: "curl\\s+.*\\|\\s*(sh|bash|zsh)"
    file_types: ["*.sh", "*.bash"]
```

### Scoring Algorithm

Each detected pattern contributes to the risk score based on severity:
- Critical: +30 points
- High: +15 points
- Medium: +5 points
- Low: +2 points

Score is capped at 100. Multiple findings of the same pattern don't stack beyond 2x.

## Excluded From Scope

- No web UI in v0.1–v0.3
- No hosted registry infrastructure (teams use Git repos)
- No plugin/extension system for Coach itself
- No lockfile or dependency resolution (expected to come from the spec)
- No semantic versioning enforcement (defers to spec evolution)
- No auto-update of Coach binary (users update via package manager or download)
