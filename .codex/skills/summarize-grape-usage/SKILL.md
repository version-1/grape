---
name: summarize-grape-usage
description: Explain how to use the grape Git-worktree CLI. Use when answering questions about grape commands, worktree setup, reset configuration, delegated git-worktree commands, or removal safety.
---

# Summarize Grape Usage

Explain `grape` usage accurately and concisely for the user's task.

## Required source

Before answering, read the canonical [REFERENCE.md](../../../REFERENCE.md). Treat it as the source of truth for commands, flags, configuration, exit codes, and safety behavior. Do not rely on memory or duplicate its detailed specification in this skill.

## Answer workflow

1. Identify whether the user needs to inspect worktrees, find a branch, add or delegate a Git worktree operation, remove worktrees, reset from configuration, or check version/help.
2. Give the smallest applicable command example. Use `go run ./cmd/grape ...` for a repository checkout, or `grape ...` when the binary is installed.
3. For reset questions, describe the relevant `grape.json` fields and config lookup behavior from `REFERENCE.md`. Include a minimal JSON example only when it helps the user.
4. For unrecognized grape subcommands, explain that grape delegates them to `git worktree` and give the appropriate `grape` invocation.
5. State important prerequisites, outcomes, and limitations that apply to the requested command.

## Safety

Treat `grape remove` and `grape reset` as destructive. Clearly say that they delete matching or non-default worktrees and associated local branches, show targets, and require `y` or `yes` confirmation. Remind the user to verify the repository, prefix or regular expression, and reset configuration before proceeding. Do not imply that the main working tree or its checked-out branch can be deleted.
