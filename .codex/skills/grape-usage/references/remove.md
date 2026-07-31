# `grape remove`

## `grape remove <prefix>`

Previews and, after confirmation, removes worktrees whose paths are the prefix itself or are under that prefix. It also deletes their associated local branches with `git branch -D`.

```sh
grape remove ../worktrees
```

The prefix is normalized to an absolute path before comparison. Before deleting anything, the command displays the affected worktrees and branches and continues only after `y` or `yes` confirmation. The main working tree is never removed.

## `grape remove --regex <pattern>`

Previews and, after confirmation, removes worktrees whose paths match the regular expression. It also deletes their associated local branches with `git branch -D`.

```sh
grape remove --regex 'repo-feature-.+'
grape remove -r 'repo-feature-.+'
```

The main working tree is never removed. An invalid regular expression exits with code `2`.
