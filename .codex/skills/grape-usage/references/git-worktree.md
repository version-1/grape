# Delegated `git worktree` Commands

`grape` handles these commands directly: no arguments (equivalent to `list`), `list`,
`help`/`--help`/`-h`, `version`, `init`, `branch`, `remove`, `reset`, and `push`.

All other arguments are delegated directly to `git worktree`.

```sh
grape prune
grape list --porcelain
```
