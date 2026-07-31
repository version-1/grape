# grape Reference

`grape` is a Go CLI for `git worktree` workflows.

It delegates standard `git worktree` operations and adds a readable worktree list, reverse branch lookup, prefix or regular-expression removal, and configuration-driven reset.

## Running grape

```sh
go run ./cmd/grape list
```

Build `grape` with `make build`. The executable entry point is `./cmd/grape`.

## Commands

### `grape help`

Displays available commands, usage, and config lookup order.

```sh
go run ./cmd/grape help
go run ./cmd/grape --help
go run ./cmd/grape -h
```

### `grape list`

Displays worktree paths, branches, and HEAD revisions in a colored table.

```sh
go run ./cmd/grape list
```

Running `grape` with no arguments behaves the same as `grape list`.

### `grape branch <branch-name>`

Displays worktrees that reference the specified local branch.

```sh
go run ./cmd/grape branch feature/example
go run ./cmd/grape branch refs/heads/feature/example
```

The `refs/heads/` prefix is optional. If no matching worktree exists, the command exits with code `1`.

### `grape remove <prefix>`

Previews and, after confirmation, removes worktrees whose paths are the prefix itself or are under that prefix. It also deletes their associated local branches with `git branch -D`.

```sh
go run ./cmd/grape remove ../worktrees
```

The prefix is normalized to an absolute path before comparison. Before deleting anything, the command displays the affected worktrees and branches and continues only after `y` or `yes` confirmation. The main working tree is never removed.

### `grape remove --regex <pattern>`

Previews and, after confirmation, removes worktrees whose paths match the regular expression. It also deletes their associated local branches with `git branch -D`.

```sh
go run ./cmd/grape remove --regex 'repo-feature-.+'
go run ./cmd/grape remove -r 'repo-feature-.+'
```

The main working tree is never removed. An invalid regular expression exits with code `2`.

### `grape reset`

Reads a configuration file, removes non-default worktrees and local branches, and then creates the configured worktrees.

```sh
go run ./cmd/grape reset
go run ./cmd/grape reset --config grape.json
go run ./cmd/grape reset -c grape.json
```

The default config file is `grape.json`. When `--config` is omitted, paths are resolved in this order:

1. `./grape.json`
2. `$GRAPE_HOME/grape.json`
3. `~/.grape/grape.json` when `GRAPE_HOME` is unset

### `grape init`

Creates `~/.grape/grape.json` from `grape.example.json` in the current directory.

```sh
grape init
```

Set `GRAPE_HOME` to create `$GRAPE_HOME/grape.json` instead. The command creates its destination directory when necessary and refuses to overwrite an existing config file.

### Migrating from gw

This release does not automatically read the former `gw` locations. Rename `gw.json` to `grape.json`, replace `GW_HOME` with `GRAPE_HOME`, and move `~/.gw` to `~/.grape`. Pass `--config` to use a config file at another path.

`reset` is destructive. It deletes all non-default worktrees and local branches, but never the main working tree or the branch checked out there. Before deleting anything, it lists the targets and continues only after `y` or `yes` confirmation. Verify the target repository and configuration file before running it.

### `grape version`

Displays the version and commit hash embedded at build time.

```sh
grape version
```

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

The branch excluded from deletion. When omitted, `grape` detects it from `origin/HEAD`.

### `worktrees[].path`

The path of the worktree to create. Paths must resolve to unique locations, so values such as `worktree`, `./worktree`, and that same path expressed absolutely cannot all be configured.

### `worktrees[].branch`

The local branch name to create. The `refs/heads/` prefix is optional, and each configured branch must be unique.

### `worktrees[].start_point`

The value supplied as `<start_point>` to `git worktree add -B <branch> <path> <start_point>`. When omitted, `default_branch` is used.

## Unknown Commands

Arguments other than `list`, `branch`, `remove`, and `reset` are delegated directly to `git worktree`.

```sh
go run ./cmd/grape prune
go run ./cmd/grape list --porcelain
```

## Package Layout

```text
cmd/grape/              # entry point
internal/app/        # CLI dispatch and command orchestration
internal/config/     # reset config schema and validation
internal/ui/         # colored output and worktree table formatting
internal/worktree/   # Git adapter, worktree parser, and matcher
```
