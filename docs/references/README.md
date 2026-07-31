# grape Reference

`grape` is a Git wrapper designed for use by coding agents.

It provides agent-friendly `git worktree` workflows, including a readable worktree list, reverse branch lookup, prefix or regular-expression removal, configuration-driven reset, and delegation of standard `git worktree` operations.

## Running grape

```sh
grape list
```

Build `grape` with `make build`. The executable entry point is `./cmd/grape`.

## Commands

- [`grape help`](help.md): display available commands, usage, and config lookup order.
- [`grape list`](list.md): display worktree paths, branches, and HEAD revisions.
- [`grape branch`](branch.md): find worktrees that reference a local branch.
- [`grape remove`](remove.md): remove matching worktrees and their local branches.
- [`grape reset`](reset.md): recreate configured worktrees after removing non-default worktrees and local branches.
- [`grape init`](init.md): create an initial configuration file and migrate from `gw`.
- [`grape version`](version.md): display build version information.
- [`git worktree` commands](git-worktree.md): delegated commands not implemented by `grape`.

## Package Layout

```text
cmd/grape/              # entry point
internal/app/            # CLI dispatch and command orchestration
internal/config/         # reset config schema and validation
internal/ui/             # colored output and worktree table formatting
internal/worktree/       # Git adapter, worktree parser, and matcher
```
