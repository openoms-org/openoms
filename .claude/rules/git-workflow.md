---
description: Git workflow rules for all agents and sessions
globs: "**/*"
---

# Git Workflow Rules

## NEVER push directly to main

This is a hard, non-negotiable rule for ALL agents and sessions:

1. **Always create a feature branch** before committing
2. **Always open a PR** via `gh pr create`
3. **Never bypass branch protection** — main requires PRs with 6 passing CI checks
4. **Never use `--force` push** unless explicitly asked by the user

## Commit messages

- No `Co-Authored-By` lines
- No "Generated with Claude Code" or similar AI attribution
- Keep messages concise, focused on the "why"

## PR descriptions

- No "Generated with Claude Code" or similar attribution
- No references to the private enterprise repo
