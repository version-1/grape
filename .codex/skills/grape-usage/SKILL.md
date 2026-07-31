---
name: grape-usage
description: Explain how to use the grape Git-worktree CLI. Use when answering questions about grape commands, worktree setup, reset configuration, delegated git-worktree commands, or removal safety.
---

# Grape Usage

Explain `grape` usage accurately and concisely for the user's task.

## Required sources

Before answering, read the relevant canonical command reference in [`references/`](references/README.md). Treat the relevant reference page as the source of truth for commands, flags, configuration, exit codes, and safety behavior. Do not rely on memory or duplicate its detailed specification in this skill.

- For command discovery and shared running or package information, read [`README.md`](references/README.md).
- For help, list, branch, remove, reset, init, and version questions, read [`help.md`](references/help.md), [`list.md`](references/list.md), [`branch.md`](references/branch.md), [`remove.md`](references/remove.md), [`reset.md`](references/reset.md), [`init.md`](references/init.md), or [`version.md`](references/version.md), respectively.
- For unknown or delegated commands, read [`git-worktree.md`](references/git-worktree.md).

## Answer workflow

1. Identify whether the user needs to inspect worktrees, find a branch, initialize configuration, add or delegate a Git worktree operation, remove worktrees, reset from configuration, or check version/help.
2. Give the smallest applicable command example using `grape ...`.
3. For reset questions, describe the relevant `grape.json` fields and config lookup behavior from [`reset.md`](references/reset.md). Include a minimal JSON example only when it helps the user.
4. For initialization questions, use the `grape init` behavior in [`init.md`](references/init.md).
5. For unrecognized grape subcommands, explain that grape delegates them to `git worktree` and give the appropriate `grape` invocation.
6. State important prerequisites, outcomes, and limitations that apply to the requested command.

## Safety

Treat `grape remove` and `grape reset` as destructive. Explain their scopes separately: `remove` deletes the matching worktrees and their associated local branches, while `reset` deletes non-default worktrees and local branches but protects the default branch and the branch checked out in the main working tree. Both commands show targets and require `y` or `yes` confirmation. Remind the user to verify the repository, prefix or regular expression, and reset configuration before proceeding. Do not imply that the main working tree or its checked-out branch can be deleted.
