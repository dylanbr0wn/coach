# Coach

Develop, test, and manage AI agent skills from the command line.

## What it does

Coach is the developer experience layer for the [Agent Skills](https://agentskills.io) ecosystem. It handles everything between writing a skill and running it: scaffolding, linting, security scanning, installation, and status tracking. Coach works across Claude Code, Cursor, Codex, and Copilot, auto-detecting which agents you have installed and managing skills for all of them.

## Installation

Install with Go:

```bash
go install github.com/dylanbr0wn/coach@latest
```

Or build from source:

```bash
git clone https://github.com/dylanbr0wn/coach.git
cd coach
go build -o coach .
```

## Quick Start

```bash
# Scaffold a new skill with an interactive form
coach init skill

# Validate a skill against the spec
coach lint ./my-skill/

# Run a deep security scan
coach scan ./my-skill/

# Install a skill (scans automatically, installs to all detected agents)
coach install owner/repo

# See what's installed across your agents
coach status
```

## Commands

| Command | Description |
|---|---|
| `coach init skill` | Scaffold a new skill with interactive prompts |
| `coach lint [path]` | Validate spec compliance, structure, and quality |
| `coach scan <path\|source>` | Deep security analysis with risk scoring |
| `coach install <source>` | Install a skill from GitHub, URL, or local path |
| `coach status` | Dashboard of detected agents and installed skills |
| `coach preview <path>` | Render a SKILL.md in the terminal |
| `coach update-rules` | Fetch latest security patterns and agent registry |

## Security Scanning

`coach scan` analyzes skills for three categories of issues:

- **Prompt injection** -- instruction overrides, hidden unicode, encoded payloads, external URL references
- **Dangerous scripts** -- shell piping, credential access, network exfiltration, out-of-scope filesystem operations
- **Quality warnings** -- overly broad descriptions, missing permission boundaries, conflicting instructions

Each skill receives a risk score from 0 to 100:

| Score | Level | Meaning |
|---|---|---|
| 0-25 | LOW | Safe to install |
| 26-50 | MEDIUM | Review warnings before installing |
| 51-75 | HIGH | Manual review recommended |
| 76-100 | CRITICAL | Blocked from install by default |

## Agent Support

Coach auto-detects the following agents by checking for their config directories:

- **Claude Code** -- `~/.claude/skills/`
- **Cursor** -- `~/.cursor/rules/`
- **Codex** -- `~/.codex/skills/`
- **Copilot**

When installing a skill, Coach symlinks it into each detected agent's skill directory (or use `--agent <name>` to target one).

## Configuration

Coach stores its configuration in `~/.coach/`:

```
~/.coach/
  config.yaml       # Global settings (default agents, trusted sources, scan preferences)
  trust/             # Signed trust records for vetted skills
  cache/             # Downloaded skill repos, scan results
  rules/             # Remote-updated security patterns and agent registry
  installed.yaml     # Provenance records for all installed skills
```

Run `coach update-rules` to fetch the latest security patterns and agent registry without upgrading the binary.

## Roadmap

- **v0.2** -- Scenario-based testing (`coach test`), setup diagnostics (`coach doctor`), auto-fix (`coach lint --fix`)
- **v0.3** -- Team registries, shared trust policies, cross-team audit (`coach team` commands)

## Version

Current: `0.1.0-dev`

## License

MIT
