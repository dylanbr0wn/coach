# Skill Editing & Authoring — Design Spec

**Date:** 2026-03-29
**Status:** Draft
**Scope:** v0.2

## Problem

Coach can scaffold a skill (`coach init`) but cannot edit one. The SKILL.md body — the actual skill instructions — is left as a skeleton after creation. Without authoring capabilities, coach is a scaffolder, not an authoring tool. This is a non-starter.

## Goals

1. Let developers edit skill content using their preferred editor
2. Provide LLM-assisted authoring for generating and refining skill bodies
3. Establish a clear storage model with local and global skill scopes
4. Automate distribution of skills to agent directories via symlinks

## Non-Goals

- Direct LLM API integration (future, if CLI shelling proves limiting)
- Multi-LLM CLI testing beyond Claude Code at launch
- Skill versioning or publishing
- `coach lint --fix` auto-repair

---

## Skill Storage Model

### Directory Structure

```
~/.coach/
├── config.yaml          # global config (distribution targets, LLM CLI, etc.)
├── skills/              # global skills (source of truth)
│   ├── code-reviewer/
│   │   └── SKILL.md
│   └── tdd-helper/
│       └── SKILL.md
├── cache/               # (existing) fetched remote skills
└── installed.yaml       # (existing) install provenance

./project-root/
└── .coach/
    └── skills/          # local skills (project-scoped)
        └── deploy-check/
            └── SKILL.md
```

### Resolution Order

When resolving a skill by name, coach searches:

1. **Local** — walk up from cwd to find `.coach/skills/<name>/SKILL.md`
2. **Global** — `~/.coach/skills/<name>/SKILL.md`
3. **Error** — skill not found, suggest `coach init skill` or `coach generate <name>`

The `--global` / `-g` and `--local` / `-l` flags force resolution to a specific scope.

### Distribution

Skills are symlinked to agent directories. Coach owns the source of truth; agent dirs are distribution targets.

Configured in `~/.coach/config.yaml`:

```yaml
distribute-to:
  - claude    # → ~/.claude/skills/
  - cursor    # → ~/.cursor/rules/
```

Symlinks mean edits in coach's directory are immediately reflected in agent dirs. Distribution is triggered after successful lint on any write operation (init, edit, generate), with a user confirmation prompt.

---

## Command: `coach edit`

### Usage

```
coach edit <skill-name> [flags]
```

### Behavior

1. Resolve skill name to SKILL.md path (local then global)
2. Open `$EDITOR` (fall back to `vi` if unset) with the SKILL.md path
3. On editor close, run `coach lint` against the skill
4. If lint finds issues: display them and prompt `"Re-open to fix? [Y/n]"`
5. If lint passes: display success, prompt to distribute if configured

### Flags

| Flag | Description |
|------|-------------|
| `--global`, `-g` | Force resolution to global skills only |
| `--local`, `-l` | Force resolution to local skills only |

### Edge Cases

- **Skill doesn't exist** — error: `"skill 'foo' not found. Did you mean 'coach init skill' or 'coach generate foo'?"`
- **`$EDITOR` not set, `vi` unavailable** — error with message to set `$EDITOR`
- **No changes made** — skip lint, print `"No changes detected"`

---

## Command: `coach generate`

### Usage

```
coach generate <skill-name> [flags]
coach generate <skill-name> --prompt "description of what this skill should do"
```

### Interactive Mode (default, no `--prompt`)

1. Resolve skill name. If it exists, load current SKILL.md as context. If it doesn't exist, create the skill directory (no interactive form — the LLM will generate the full SKILL.md including frontmatter).
2. Build a system prompt that includes:
   - The SKILL.md format spec (frontmatter schema, body conventions)
   - The current SKILL.md content (if editing existing skill)
   - Instruction to output a complete, valid SKILL.md
   - Reference example (the fleshed-out `skill-coach/SKILL.md`)
3. Shell out to the configured LLM CLI (default: `claude`) in interactive mode, passing the system prompt as context.
4. User converses with the LLM to author/refine the skill.
5. On session end, coach reads back the result, writes SKILL.md.
6. Run `coach lint` automatically, display results.

### Single-Shot Mode (`--prompt`)

1. Same resolution and system prompt setup.
2. Shell out to `claude --print -p "..."` with the system prompt + user instruction.
3. Parse the LLM output, write SKILL.md.
4. Run `coach lint`, display results.
5. Show diff of what changed and prompt `"Accept changes? [Y/n]"`.

### Flags

| Flag | Description |
|------|-------------|
| `--prompt`, `-p` | Single-shot mode with inline instruction |
| `--global`, `-g` | Create/edit in global skills |
| `--local`, `-l` | Create/edit in local skills (default when in a project with `.coach/`) |
| `--cli` | Override LLM CLI for this invocation |

### System Prompt Design

The system prompt is the key differentiator. It encodes coach's knowledge of what makes a good skill:

- SKILL.md frontmatter schema and validation rules
- Body structure best practices (trigger conditions, instructions, constraints)
- Security considerations (no secrets, no destructive defaults)
- Embedded reference skill (`skill-coach/SKILL.md`) as a well-formed example

### Edge Cases

- **LLM CLI not found** — error: `"claude CLI not found. Install it or configure a different CLI: coach config set llm-cli <command>"`
- **LLM output not valid SKILL.md** — show raw output, ask user to retry or edit manually
- **User declines changes (single-shot)** — no write, skill unchanged

---

## Command: `coach config`

### Usage

```
coach config set <key> <value>
coach config get <key>
```

### Supported Keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `distribute-to` | comma-separated list | (empty) | Agent targets for symlinking |
| `llm-cli` | string | `claude` | LLM CLI command for `coach generate` |
| `default-scope` | `global` or `local` | `global` | Default scope for new skills outside a project |

### Config File

Global: `~/.coach/config.yaml`
Local override: `.coach/config.yaml` in project root

```yaml
distribute-to:
  - claude
  - cursor
llm-cli: claude
default-scope: global
```

---

## Command: `coach sync`

### Usage

```
coach sync [flags]
```

### Behavior

Re-symlinks all managed skills to configured agent directories. Useful after config changes, adding new agents, or if symlinks break.

1. Read distribution targets from config.
2. For each managed skill (local + global), ensure symlink exists in each target agent directory.
3. Report what was created, updated, or already in sync.

### Flags

| Flag | Description |
|------|-------------|
| `--global`, `-g` | Sync global skills only |
| `--local`, `-l` | Sync local skills only |
| `--dry-run` | Show what would be done without making changes |

---

## Changes to Existing Commands

### `coach init`

- Add `--global` / `--local` flags to control where the skill is created
- Default to local if in a project with `.coach/`, global otherwise
- After scaffolding, prompt to distribute if configured
- Update getting started text to reflect new workflow: `coach init skill` → `coach edit <name>` or `coach generate <name>`

---

## Help Text & Examples

Every command must have clear help text with realistic examples. The current help is sparse and has caused confusion during QA (e.g., `coach install claude` shown in getting started text but doesn't work).

### Principles

- **Show the full workflow** — help text should show what comes before and after a command, not just the command in isolation
- **Use realistic examples** — real skill names, real flags, real output snippets
- **Explain concepts inline** — first mention of "global" vs "local" skills should briefly explain the distinction
- **Getting started section** — update the root help to walk through: `coach init skill` → `coach edit <name>` or `coach generate <name>` → `coach sync`

### Per-Command Examples

**`coach edit`:**
```
Examples:
  coach edit code-reviewer          # Open in $EDITOR, lint on save
  coach edit code-reviewer -g       # Edit the global version
  coach edit deploy-check -l        # Edit the local (project) version
```

**`coach generate`:**
```
Examples:
  coach generate code-reviewer                              # Interactive: chat with LLM to author the skill
  coach generate code-reviewer -p "help review Go code"     # Single-shot: generate from a prompt
  coach generate new-skill -g                               # Create and author a new global skill
  coach generate my-skill --cli codex                       # Use a different LLM CLI
```

**`coach config`:**
```
Examples:
  coach config set distribute-to claude,cursor    # Distribute skills to Claude and Cursor
  coach config set llm-cli claude                 # Set default LLM CLI
  coach config get distribute-to                  # Show current distribution targets
```

**`coach sync`:**
```
Examples:
  coach sync                # Symlink all skills to configured agents
  coach sync --dry-run      # Preview what would be linked
  coach sync -g             # Sync global skills only
```

### Root Help Update

The grouped help template in `cmd/root.go` needs updated categories and getting started text:

```
Getting Started:
  1. coach init skill                  Create a new skill
  2. coach edit <name>                 Write the skill content (or use coach generate)
  3. coach lint <path>                 Validate the skill
  4. coach config set distribute-to claude   Configure where skills are distributed
  5. coach sync                        Symlink skills to your agents
```

---

## Prerequisites

1. **Skill resolution layer** — new `internal/resolve` package to search local → global
2. **Flesh out `skill-coach/SKILL.md`** — must be a proper reference skill for the generate system prompt
3. **Distribution logic** — new `internal/distribute` package for symlinking to agent dirs
4. **Config management** — extend `internal/config` to support new fields

## Implementation Order

1. Config + storage model (foundation)
2. Skill resolution layer (`internal/resolve`)
3. Distribution / symlinking (`internal/distribute`)
4. `coach edit` command
5. Reference `skill-coach/SKILL.md`
6. `coach generate` command
7. `coach sync` command
8. Update `coach init` for new storage model
9. Update help text and examples across all commands
