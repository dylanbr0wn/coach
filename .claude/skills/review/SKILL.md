---
name: review
description: "PR review, fix, and merge workflow. Use when the user wants to review a pull request, fix issues found in a PR, check PR quality, or run `/review <number>`. Reads the full PR diff, reviews for bugs/security/quality, categorizes findings by severity, fixes critical and important issues, runs build/lint/test checks, pushes fixes, and asks before merging. Also use when the user says things like 'review PR 42', 'check this PR', 'look at the PR', or 'merge PR'."
---

# PR Review & Fix

A complete review-to-merge workflow for pull requests. Invoked with a PR number.

## Step 1: Understand the PR

Run these in parallel:

```bash
gh pr view <number> --json title,body,baseRefName,headRefName,author
gh pr diff <number>
gh pr checks <number>
```

Read the PR title, description, and full diff. Then **read every changed file in full** — not just diff hunks — so you understand the surrounding context before judging any change.

## Step 2: Check out the PR branch

```bash
gh pr checkout <number>
```

Verify you're on the correct branch before making any changes.

## Step 3: Review and categorize

Examine every changed line. Look for:

- **Bugs and logic errors** — incorrect behavior, off-by-one, nil/null dereference, race conditions, incorrect error handling
- **Security vulnerabilities** — injection, XSS, hardcoded secrets, command injection, OWASP top 10
- **API contract breaks** — unintended changes to function signatures or return types that break callers
- **Stale references** — removed variables, props, or refs still referenced elsewhere (grep to confirm)
- **Code quality** — unclear naming, unnecessary complexity, missing validation at system boundaries

Assign each finding a severity:

| Severity | Criteria | Action |
|----------|----------|--------|
| **Critical** | Bugs, security issues, data loss, broken functionality | Fix |
| **Important** | API misuse, stale references, missing boundary validation, perf issues | Fix |
| **Minor** | Style nits, naming preferences, optional improvements | Report only |

### Present findings before fixing

Show the user a structured summary so they know what's coming:

```
## PR Review: #<number> — <title>

### Critical (N)
- description of issue (file:line)

### Important (N)
- description of issue (file:line)

### Minor (N)
- description (file:line)

Fixing N critical and N important issues now.
```

If there are zero critical and zero important issues, skip to Step 5 (build checks) — don't make changes just to make changes.

## Step 4: Fix critical and important issues

For each fix:

- **Smallest possible change.** Don't refactor surrounding code, change function signatures, or modify return types unless the issue specifically demands it.
- **Grep after removals.** After removing any variable, prop, import, or ref, grep the entire codebase for remaining references and clean them up.
- **Don't fix minor issues.** Leave them for the PR author — these are suggestions, not blockers.

## Step 5: Run build and lint checks

Run the project's full verification suite from CLAUDE.md. For this project:

```bash
# Go
go vet ./internal/...
go test ./internal/... -race -timeout 60s
go mod tidy && git diff --exit-code go.mod go.sum

# Frontend
cd frontend && pnpm build
cd frontend && pnpm lint
cd frontend && pnpm format:check
```

If any check fails, fix the root cause and rerun until everything passes. Don't skip or ignore failures.

## Step 6: Commit and push

Commit fixes with a descriptive message:

```
fix(<scope>): address PR review findings

- <one line per fix>

Co-Authored-By: Claude <noreply@anthropic.com>
```

Push to the PR branch:

```bash
git push
```

## Step 7: Report and ask before merging

Present the final status:

```
## Merge Readiness: #<number>

- N critical issues fixed
- N important issues fixed
- N minor issues noted (not fixed)
- All checks passing / N checks failing

Ready to merge. Shall I merge this PR?
```

**Wait for explicit user confirmation.** Never merge without the user saying yes.

When confirmed:

```bash
gh pr merge <number> --squash --delete-branch
```
