---
name: git-commit
description: 'Execute git commit with conventional commit message analysis, intelligent staging, and message generation. Use when user asks to commit changes, create a git commit, or mentions "/commit". Supports: (1) Auto-detecting type and scope from changes, (2) Generating conventional commit messages from diff, (3) Interactive commit with optional type/scope/description overrides, (4) Intelligent file staging for logical grouping'
license: MIT
allowed-tools: Bash
---

# Git Commit with Conventional Commits

## Overview

Create standardized, semantic git commits using the Conventional Commits
specification. Analyze the actual diff to determine appropriate type, scope, and
message.

## Conventional Commit Format

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Commit Types

| Type | Purpose |
| --- | --- |
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting or style only, with no logic change |
| `refactor` | Code refactor with no feature or fix |
| `perf` | Performance improvement |
| `test` | Add or update tests |
| `build` | Build system or dependency changes |
| `ci` | CI or config changes |
| `chore` | Maintenance or miscellaneous work |
| `revert` | Revert a previous commit |

## Breaking Changes

```text
# Exclamation mark after type or scope
feat!: remove deprecated endpoint

# BREAKING CHANGE footer
feat: allow config to extend other configs

BREAKING CHANGE: extends key behavior changed
```

## Workflow

### 1. Analyze Diff

```bash
# If files are staged, use staged diff
git diff --staged

# If nothing is staged, use working tree diff
git diff

# Also check status
git status --porcelain
```

### 2. Stage Files if Needed

If nothing is staged or you want to group changes differently:

```bash
# Stage specific files
git add path/to/file1 path/to/file2

# Stage by pattern
git add *.test.*
git add src/components/*

# Interactive staging
git add -p
```

Never commit secrets such as `.env`, `credentials.json`, or private keys.

### 3. Generate Commit Message

Analyze the diff to determine:

- Type: What kind of change is this?
- Scope: What area or module is affected?
- Description: One-line summary of what changed in present tense, imperative
  mood, under 72 characters

### 4. Execute Commit

```bash
# Single line
git commit -m "<type>[scope]: <description>"

# Multi-line with body and footer
git commit -m "<type>[scope]: <description>" \
  -m "<optional body>" \
  -m "<optional footer>"
```

## Best Practices

- One logical change per commit
- Present tense: `add` not `added`
- Imperative mood: `fix bug` not `fixes bug`
- Reference issues when relevant: `Closes #123`, `Refs #456`
- Keep the description under 72 characters

## Project Constraints

- Use Conventional Commits as `type(scope): subject` when a scope is available.
- Commit subject and body must be fully in English.
- When the body lists multiple change items, format them as flat `-` bullets.
- When composing multi-line messages in the terminal, use separate `-m` flags or
  real newlines; never place literal `\n` sequences inside the message text.

```bash
git commit -m "feat(auth): tighten public auth API docs" \
  -m "- document Identity interface methods
- document TokenType constants"
```

## Copyable Short Prompts

Use these prompts when you want a fast, copy-ready trigger in day-to-day chat.

### Slash Invocations

```text
/git-commit
/git-commit staged changes
/git-commit current diff
/git-commit .github/skills
```

### Natural-Language Triggers

```text
Use git-commit for the current changes.
Use git-commit for staged files only.
Use git-commit and suggest a conventional commit message.
Use git-commit for .github/skills with scope copilot.
```

### Prompt Shaping Tips

- Say `staged files only` when the staging is already correct.
- Name the target path when only part of the working tree should be committed.
- Say `suggest type and scope` when you want auto-detection.
- Say `use scope <name>` when you already know the scope.

## Git Safety Protocol

- NEVER update git config
- NEVER run destructive commands such as `--force` or hard reset without
  explicit request
- NEVER skip hooks with `--no-verify` unless the user asks
- NEVER force push to `main` or `master`
- If commit fails due to hooks, fix the issue and create a new commit instead of
  amending silently