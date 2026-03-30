# Design: `coach list` command

**Date:** 2026-03-29
**Issue:** No way to list skills per agent — `status` shows counts but no details

## Purpose

Show installed skills per agent with enough detail to act on (path for `scan`, name for `install`/`edit`). Bridges the gap between `status` (counts only) and manual filesystem exploration.

## CLI Interface

```
coach list                        # all agents, grouped by agent
coach list --agent claude-code    # filter to one agent
coach list --format json          # JSON output for scripting
```

### Flags

| Flag       | Type   | Default | Description                          |
|------------|--------|---------|--------------------------------------|
| `--agent`  | string | `""`    | Filter to a specific agent by key    |
| `--format` | string | `table` | Output format: `table` or `json`     |

## Default Table Output

```
  Claude Code (~/.claude/skills/)

  Name              Description                    Path                           Vetted
  ──────────────────────────────────────────────────────────────────────────────────────
  my-skill          Does something useful          ~/.claude/skills/my-skill/     ✓
  untrusted-skill   Borrowed from internet         ~/.claude/skills/untrusted/    ✗

  Cursor (~/.cursor/skills/)

  Name              Description                    Path                           Vetted
  ──────────────────────────────────────────────────────────────────────────────────────
  another-skill     Another thing                  ~/.cursor/skills/another/      ✓
```

- Descriptions truncated to 40 characters in table mode
- Each agent group has a header showing agent name and skill directory

## JSON Output

```json
[
  {
    "agent": "Claude Code",
    "skill_dir": "~/.claude/skills/",
    "skills": [
      {
        "name": "my-skill",
        "description": "Does something useful",
        "path": "~/.claude/skills/my-skill/",
        "vetted": true
      }
    ]
  }
]
```

Full descriptions (not truncated) in JSON mode.

## Edge Cases

- **No installed agents:** Print message "No agents detected" and exit 0.
- **Agent has no skills:** Print agent header with "No skills installed" beneath it.
- **`--agent` not found:** Error with message listing valid agent keys.
- **SKILL.md parse failure:** Show skill name with "parse error" in description column; don't crash the whole listing.

## Implementation

### New file: `cmd/list.go`

Standard cobra command pattern. Registers via `init()` with `rootCmd.AddCommand`.

### Shared helper refactor

`listSkillDirs` currently lives as an unexported function in `cmd/status.go`. Move it to `internal/skill` as `ListSkillDirs(dir string) []string` so both `status` and `list` can use it.

### Dependencies (existing packages)

- `internal/agent` — `DetectAgents`, `InstalledAgents` for agent discovery
- `internal/skill` — `Parse` for reading SKILL.md, new `ListSkillDirs` helper
- `internal/registry` — `LoadProvenance` for vetted/unvetted status
- `internal/config` — `DefaultCoachDir` for provenance path
- `internal/ui` — `RenderTable` for table output
- `encoding/json` — for `--format json`

### Flow

1. Detect installed agents
2. If `--agent` flag set, filter to matching agent (error if not found)
3. Load provenance from `~/.coach/installed.yaml`
4. For each agent:
   a. List skill directories via `ListSkillDirs`
   b. Parse each skill via `skill.Parse`
   c. Check provenance for vetted status
   d. Collect into result slice
5. Render as table (grouped by agent) or JSON

### Command grouping

Register under the "Discovery" group in the custom help template alongside `status`, `scan`, and `lint`.
