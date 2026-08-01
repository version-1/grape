# grape Reference

`grape` is a Go CLI for `git worktree` workflows.

It delegates standard `git worktree` operations and adds policy-gated current-branch rebase, guarded current-branch push, a readable worktree list, reverse branch lookup, prefix or regular-expression removal, and configuration-driven reset.

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
- [`grape push`](push.md): safely push the current symbolic branch to `origin`.
- [`grape rebase`](rebase.md): rebase the current symbolic branch when allowed by config.
- [`grape init`](init.md): create an initial configuration file.
- [`grape version`](version.md): display build version information.
- [`git worktree` commands](git-worktree.md): delegated commands not implemented by `grape`.
- [Configuration](config.md): strict decoding, discovery, and protected branch patterns.
- [Output](output.md): stderr logging and automatic color behavior.

## Package Layout

```text
cmd/grape/              # entry point
internal/app/            # CLI dispatch and command orchestration
internal/color/          # output-specific terminal and NO_COLOR policy
internal/config/         # strict shared config schema and command validation
internal/logging/        # grape-generated stderr diagnostics
internal/ui/             # colored output and worktree table formatting
internal/worktree/       # Git, push, and rebase adapter, worktree parser, and matcher
```
