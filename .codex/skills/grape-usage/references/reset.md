# `grape reset`

Reads a configuration file, removes non-default worktrees and local branches, and then creates the configured worktrees.

```sh
grape reset
grape reset --yes
grape reset -y
grape reset --config grape.json
grape reset -c grape.json
grape reset -y -c grape.json
```

The default config file is `grape.json`. When `--config` is omitted, paths are resolved in this order:

1. `./grape.json`
2. `$GRAPE_HOME/grape.json`
3. `~/.grape/grape.json` when `GRAPE_HOME` is unset

The selected file is decoded strictly before Git inspection. Unknown fields, malformed JSON, invalid values, and unreadable files are errors. Reset additionally requires a non-empty, valid `worktrees` list; a push-only config is valid globally but not sufficient for reset.

`reset` is destructive. It deletes all non-default worktrees and local branches, but never the main working tree or the branch checked out there. Before deleting anything, it lists the targets and the counts of worktrees to remove and add, then continues only after `y` or `yes` confirmation. Use `--yes` or `-y` to skip the prompt; the deletion preview is still printed. Verify the target repository and configuration file before running it.

Reset prints a phase message before removing worktrees, deleting local branches, and adding worktrees. It suppresses standard output from those Git operations while preserving Git diagnostics on standard error. Reset removes each target worktree with `git worktree remove --force` whether or not `--yes` is used, and deletes target branches with `git branch -D`. After a successful reset, it prints the exact output from `git branch`. Failure during deletion or creation stops the reset without printing a final branch list. Failure to retrieve the final branch list also makes the command fail; completed Git changes are not rolled back.

## Reset Configuration

```json
{
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  },
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
