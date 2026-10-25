# `grape start-*`

Creates a prefixed local branch in the current worktree after synchronizing the selected default branch with `origin`.

```sh
grape start-feat add-start-command
grape start-docs document-start-command
grape start-refactor split-branch-workflow
grape start-test start-command
grape sf add-start-command
```

The commands create and check out `feat/<name>`, `docs/<name>`, `refactor/<name>`, and `test/<name>`, respectively. Each validates the complete branch name and rejects a name that matches a configured protected branch.

`sf`, `sd`, `sr`, and `st` are aliases for `start-feat`, `start-docs`, `start-refactor`, and `start-test`.

Before creating the branch, grape requires a clean current worktree, fetches the selected default branch explicitly from `origin`, checks out the configured `default_branch` or the branch advertised by `origin` as `HEAD`, and fast-forwards it with `git merge --ff-only origin/<default>`. It then creates the prefixed branch from that synchronized local branch. If the default branch is absent locally, grape creates it from `origin/<default>` without setting an upstream.

The command refuses to overwrite an existing local branch. Git rejects unsafe checkouts, including a default branch checked out in another worktree or a default branch that has local commits and cannot fast-forward. The default branch created when it is absent locally does not receive an upstream; each start command explicitly reads `origin`. Fetch, checkout, or merge failures stop the command; grape does not force-update, reset, rebase, or roll back completed Git changes.
