package llm

import (
	_ "embed"
	"strings"
)

// referenceSkill is the canonical skill-coach/SKILL.md embedded at build time.
// Run "go generate ./internal/llm/" to refresh after editing skill-coach/SKILL.md.
//
//go:generate cp ../../skill-coach/SKILL.md reference_skill.md
//go:embed reference_skill.md
var referenceSkill string

// BuildSystemPrompt constructs a system prompt for LLM-assisted skill authoring.
// If referenceOverride is non-empty it is used instead of the embedded reference skill.
// If existingContent is non-empty the prompt asks the LLM to refine the existing skill;
// otherwise it asks the LLM to create a new skill from scratch.
func BuildSystemPrompt(existingContent, referenceOverride string) string {
	var b strings.Builder

	b.WriteString("You are a skill authoring assistant. Your job is to help write and refine SKILL.md files for AI agent skills.\n\n")

	b.WriteString("## SKILL.md Format Specification\n\n")
	b.WriteString("A valid SKILL.md consists of YAML frontmatter followed by a Markdown body.\n\n")
	b.WriteString("### Frontmatter Schema\n\n")
	b.WriteString("```yaml\n")
	b.WriteString("---\n")
	b.WriteString("name: <string>          # required: lowercase alphanumeric with hyphens only, max 64 chars\n")
	b.WriteString("description: <string>   # required: what the skill does, max 1024 chars\n")
	b.WriteString("license: <string>       # optional: MIT, Apache-2.0, or ISC\n")
	b.WriteString("allowed-tools:          # optional: list of tools the skill needs\n")
	b.WriteString("  - Read\n")
	b.WriteString("  - Write\n")
	b.WriteString("  - Bash\n")
	b.WriteString("---\n")
	b.WriteString("```\n\n")
	b.WriteString("### Field Rules\n\n")
	b.WriteString("- **name**: lowercase letters, digits, and hyphens only; no spaces; max 64 characters; required\n")
	b.WriteString("- **description**: plain text summary of the skill's purpose; max 1024 characters; required\n")
	b.WriteString("- **license**: if specified, must be one of: MIT, Apache-2.0, ISC\n")
	b.WriteString("- **allowed-tools**: list only the tools strictly necessary for the skill to operate\n\n")
	b.WriteString("### Body Best Practices\n\n")
	b.WriteString("The body should be clear, actionable Markdown. Include these sections:\n\n")
	b.WriteString("1. **When to Use** — describe the trigger conditions; when should an agent activate this skill?\n")
	b.WriteString("2. **Instructions** — numbered, step-by-step guidance for the agent to follow\n")
	b.WriteString("3. **Constraints** — explicit rules about what the agent must NOT do\n\n")
	b.WriteString("Keep instructions concrete and imperative. Avoid vague language. Each step should be independently executable.\n\n")
	b.WriteString("### Security Rules\n\n")
	b.WriteString("- Never include secrets, API keys, tokens, or credentials in a skill\n")
	b.WriteString("- Never include destructive commands (rm -rf, DROP TABLE, etc.) without explicit user confirmation steps\n")
	b.WriteString("- Scope allowed-tools to the minimum set needed — do not grant Bash if only Read is required\n")
	b.WriteString("- Do not reference external URLs that could change or contain malicious content\n\n")

	refSkill := referenceSkill
	if referenceOverride != "" {
		refSkill = referenceOverride
	}
	b.WriteString("## Reference Skill Example\n\n")
	b.WriteString("The following is a well-formed example skill for your reference:\n\n")
	b.WriteString("```\n")
	b.WriteString(refSkill)
	b.WriteString("\n```\n\n")

	if existingContent != "" {
		b.WriteString("## Current Skill Content\n\n")
		b.WriteString("```\n")
		b.WriteString(existingContent)
		b.WriteString("\n```\n\n")
		b.WriteString("Help the user refine and improve this skill.\n\n")
	} else {
		b.WriteString("Help the user create a new skill from scratch.\n\n")
	}

	b.WriteString("When outputting the final SKILL.md, output ONLY the file content (frontmatter + body), no surrounding explanation or code fences.")

	return b.String()
}
