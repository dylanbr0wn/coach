# UX Overhaul Design — Issue #8

## Summary

Bundle UX improvements into three PRs that make coach feel like a cohesive product. Foundation work (output consistency) lands first, then onboarding flow, then interactive polish.

## PR 1: Output Consistency (Foundation)

### New `ui` Helpers

Add to `internal/ui/`:

**`ui.Success(msg string) string`** — returns `✓ msg` styled green. Replaces ad-hoc `ui.SuccessStyle.Render("✓")` + `fmt.Printf` patterns.

**`ui.Warn(msg string) string`** — returns `⚠ msg` styled yellow.

**`ui.Error(msg, suggestion string) string`** — returns:
```
✗ msg
  → suggestion
```
When suggestion is empty, omits the second line. Styled red.

**`ui.NextStep(cmd, desc string) string`** — returns a dimmed gray hint:
```
Next: coach <cmd> — <desc>
```

**`ui.WithSpinner(msg string, fn func() error) error`** — wraps a function with an animated spinner from `charmbracelet/bubbles/spinner`. Uses the dot style. Displays `msg` next to the spinner while running. Returns the function's error.

### Spinner Targets

These operations get a spinner:
- `generate` — LLM call
- `install` — fetch from source + scan
- `update-rules` — fetch remote patterns
- `scan` — when scanning all managed skills (not single-file)

### Command Audit

Migrate every command to use the new helpers:
- All success messages → `ui.Success()`
- All error messages with actionable suggestions → `ui.Error(msg, suggestion)`
- All warnings → `ui.Warn()`
- Remove ad-hoc inline styling patterns

No visual changes to the style system (colors, icons, lipgloss theme). This PR standardizes the *patterns*, not the *palette*.

## PR 2: Onboarding Flow

### New `coach setup` Command

Interactive first-run configuration wizard.

**Step 1 — Agent selection:** Call `agent.DetectAgents()` to find installed agents. Present a `huh.MultiSelect` of detected agents. If no agents detected, print an error with guidance on installing a supported agent.

**Step 2 — LLM CLI preference:** Present a `huh.Select` with options: claude, codex, gemini, other (text input), skip. Saved to `llm-cli` config key.

**Step 3 — Directory creation:** Create `~/.coach/skills/` and any missing agent skill directories for the selected agents.

**Step 4 — Confirmation:** Print what was configured using `ui.Success()`, then `ui.NextStep("init skill", "scaffold your first skill")`.

**Research task:** During implementation, investigate whether any agents beyond Claude Code, Cursor, Windsurf, and Copilot use skill directories that coach should detect.

**Command group:** Add to the Management group in `root.go`, positioned first.

### First-Run Detection

Commands that require config check for it and suggest setup:

- `sync` — checks `distribute-to` is set
- `install` — checks `distribute-to` is set
- `generate` — checks `llm-cli` is set

Detection uses `ui.Warn()`:
```
⚠ No agents configured for distribution
  → Run 'coach setup' to get started, or set manually with 'coach config set distribute-to claude,cursor'
```

Non-blocking: commands still work if the user passes flags or sets config manually. The suggestion is printed to stderr so it doesn't interfere with piped output.

### Next-Step Hints

Authoring commands print a gray hint after successful completion:

| Command    | Next step hint                                                    |
|------------|-------------------------------------------------------------------|
| `setup`    | `coach init skill` — scaffold your first skill                    |
| `init`     | `coach generate <name>` or `coach edit <name>` — flesh it out     |
| `generate` | `coach lint <name>` — validate before distributing                |
| `edit`     | `coach lint <name>` — validate before distributing                |
| `lint`     | `coach sync` — distribute to your agents (on success, no findings)|
| `sync`     | `coach status` — verify everything looks right                    |

Uses `ui.NextStep()`. Only printed on success. Dimmed gray so it doesn't compete with command output.

## PR 3: Interactive Defaults + Status Enhancement + Help Audit

### Interactive Skill Picker

When these commands are invoked with no arguments, present a `huh.Select` of managed skills instead of erroring:

**`coach edit`** — lists managed skills (global + local). User selects one, opens in `$EDITOR`.

**`coach generate`** — lists managed skills plus a "Create new skill" option at the top. If "Create new" is selected, runs the `init` flow first, then proceeds to generate.

**`coach preview`** — lists managed skills. User selects one, renders preview.

The skill list shows name and description. Uses `resolve.Resolver` to find managed skills in both scopes. If no managed skills exist, prints a helpful message suggesting `coach init skill`.

### Enhanced `coach status`

Two-section dashboard:

**Section 1 — Managed Skills:**
```
Managed Skills
  ~/.coach/skills/
    • my-skill         "Quick description from SKILL.md"
    • another-skill    "Another description"
  .coach/skills/
    (none)
```

Lists skills from both global and local scope. Shows name + short description parsed from SKILL.md. If a scope has no skills, shows `(none)`.

**Section 2 — Agent Sync Status:**
```
Agent Status
  Claude Code    2 skills synced, 1 unvetted
  Cursor         2 skills synced
```

Existing agent detection + skill count logic, kept as-is but formatted consistently with the new `ui` helpers.

### Help Text Audit

Every command gets:

1. **`Example` field** — 2-3 copy-pasteable examples showing real workflows:
   ```
   # Scaffold a new skill
   coach init skill

   # Scaffold in the current project
   coach init skill --local
   ```

2. **Cross-references in `Long`** — related commands mentioned at the end:
   ```
   See also: coach scan (deep security analysis), coach preview (render skill)
   ```

## Out of Scope

- Visual reskin (new colors, icons, layout) — deferred. This PR standardizes patterns; reskinning is trivial once patterns are consistent.
- `coach status` config section — use `coach config get` for that.
- Default scope in setup — power-user concept, not first-run material.
- Interactive fallbacks for `install` (needs external source) or `lint`/`scan` (already handle no-args).

## Implementation Order

1. PR 1 (output consistency) lands first — provides the helpers that PR 2 and PR 3 depend on.
2. PR 2 (onboarding) builds on the helpers.
3. PR 3 (interactive defaults + status + help) is independent of PR 2 but benefits from consistent output.
