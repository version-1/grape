---
name: summarize-grape-usage
description: Explain how to use the grape Git-worktree CLI. Use when answering questions about grape commands, worktree setup, reset configuration, delegated git-worktree commands, or removal safety.
---

# Summarize Grape Usage

Explain `grape` usage accurately and concisely for the user's task.

## Required sources

Before answering, read the relevant canonical command reference in [`docs/references/`](../../../docs/references/README.md). Treat the relevant reference page as the source of truth for commands, flags, configuration, exit codes, and safety behavior. Do not rely on memory or duplicate its detailed specification in this skill.

- For command discovery and shared running or package information, read [`docs/references/README.md`](../../../docs/references/README.md).
- For help, list, branch, remove, reset, init or `gw` migration, and version questions, read [`help.md`](../../../docs/references/help.md), [`list.md`](../../../docs/references/list.md), [`branch.md`](../../../docs/references/branch.md), [`remove.md`](../../../docs/references/remove.md), [`reset.md`](../../../docs/references/reset.md), [`init.md`](../../../docs/references/init.md), or [`version.md`](../../../docs/references/version.md), respectively.
- For unknown or delegated commands, read [`git-worktree.md`](../../../docs/references/git-worktree.md).

## Answer workflow

1. Identify whether the user needs to inspect worktrees, find a branch, initialize configuration, migrate from `gw`, add or delegate a Git worktree operation, remove worktrees, reset from configuration, or check version/help.
2. Give the smallest applicable command example using `grape ...`.
3. For reset questions, describe the relevant `grape.json` fields and config lookup behavior from [`reset.md`](../../../docs/references/reset.md). Include a minimal JSON example only when it helps the user.
4. For initialization or migration questions, use the `grape init` and `gw` migration behavior in [`init.md`](../../../docs/references/init.md); do not infer compatibility with former `gw` locations.
5. For unrecognized grape subcommands, explain that grape delegates them to `git worktree` and give the appropriate `grape` invocation.
6. State important prerequisites, outcomes, and limitations that apply to the requested command.

## Safety

Treat `grape remove` and `grape reset` as destructive. Explain their scopes separately: `remove` deletes the matching worktrees and their associated local branches, while `reset` deletes non-default worktrees and local branches but protects the default branch and the branch checked out in the main working tree. Both commands show targets and require `y` or `yes` confirmation. Remind the user to verify the repository, prefix or regular expression, and reset configuration before proceeding. Do not imply that the main working tree or its checked-out branch can be deleted.
