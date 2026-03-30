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
