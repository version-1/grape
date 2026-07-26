# gw Reference

`gw` is a Go CLI for `git worktree` workflows.

It delegates standard `git worktree` operations and adds a readable worktree list, reverse branch lookup, prefix or regular-expression removal, and configuration-driven reset.

## Running gw

```sh
cd shared/commands/gw
go run ./cmd/gw list
```

Build `gw` with `make build`. The executable entry point is `./cmd/gw`.

## Commands

### `gw help`

Displays available commands, usage, and config lookup order.

```sh
go run ./cmd/gw help
go run ./cmd/gw --help
go run ./cmd/gw -h
```

### `gw list`

Displays worktree paths, branches, and HEAD revisions in a colored table.

```sh
go run ./cmd/gw list
```

Running `gw` with no arguments behaves the same as `gw list`.

### `gw branch <branch-name>`

Displays worktrees that reference the specified local branch.

```sh
go run ./cmd/gw branch feature/example
go run ./cmd/gw branch refs/heads/feature/example
```

The `refs/heads/` prefix is optional. If no matching worktree exists, the command exits with code `1`.

### `gw remove <prefix>`

Previews and, after confirmation, removes worktrees whose paths are the prefix itself or are under that prefix. It also deletes their associated local branches with `git branch -D`.

```sh
go run ./cmd/gw remove ../worktrees
```

The prefix is normalized to an absolute path before comparison. Before deleting anything, the command displays the affected worktrees and branches and continues only after `y` or `yes` confirmation. The main working tree is never removed.

### `gw remove --regex <pattern>`

Previews and, after confirmation, removes worktrees whose paths match the regular expression. It also deletes their associated local branches with `git branch -D`.

```sh
go run ./cmd/gw remove --regex 'repo-feature-.+'
go run ./cmd/gw remove -r 'repo-feature-.+'
```

The main working tree is never removed. An invalid regular expression exits with code `2`.

### `gw reset`

Reads a configuration file, removes non-default worktrees and local branches, and then creates the configured worktrees.

```sh
go run ./cmd/gw reset
go run ./cmd/gw reset --config gw.json
go run ./cmd/gw reset -c gw.json
```

The default config file is `gw.json`. When `--config` is omitted, paths are resolved in this order:

1. `./gw.json`
2. `$GW_HOME/gw.json`
3. `~/.gw/gw.json` when `GW_HOME` is unset

`reset` is destructive. It deletes all non-default worktrees and local branches, but never the main working tree or the branch checked out there. Before deleting anything, it lists the targets and continues only after `y` or `yes` confirmation. Verify the target repository and configuration file before running it.

## Reset Configuration

```json
{
  "default_branch": "main",
  "worktrees": [
    {
      "path": ".worktrees/1",
      "branch": "worktrees/1",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/2",
      "branch": "worktrees/2",
      "start_point": "origin/main"
    }
  ]
}
```

### `default_branch`

The branch excluded from deletion. When omitted, `gw` detects it from `origin/HEAD`.

### `worktrees[].path`

The path of the worktree to create. Paths must be unique after cleaning, so values such as `worktree` and `./worktree` cannot both be configured.

### `worktrees[].branch`

The local branch name to create. The `refs/heads/` prefix is optional, and each configured branch must be unique.

### `worktrees[].start_point`

The value supplied as `<start_point>` to `git worktree add -B <branch> <path> <start_point>`. When omitted, `default_branch` is used.

## Unknown Commands

Arguments other than `list`, `branch`, `remove`, and `reset` are delegated directly to `git worktree`.

```sh
go run ./cmd/gw prune
go run ./cmd/gw list --porcelain
```

## Package Layout

```text
cmd/gw/              # entry point
internal/app/        # CLI dispatch and command orchestration
internal/config/     # reset config schema and validation
internal/ui/         # colored output and worktree table formatting
internal/worktree/   # Git adapter, worktree parser, and matcher
```
