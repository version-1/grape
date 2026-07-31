# `grape branch <branch-name>`

Displays worktrees that reference the specified local branch.

```sh
grape branch feature/example
grape branch refs/heads/feature/example
```

The `refs/heads/` prefix is optional. If no matching worktree exists, the command exits with code `1`.
