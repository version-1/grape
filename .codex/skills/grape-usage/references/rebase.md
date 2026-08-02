# `grape rebase`

Runs `git rebase` only when a strict config policy allows the current branch, passing Git rebase arguments through unchanged.

Accepted forms:

```sh
grape rebase
grape rebase [--config <path>] [<git-rebase-args>...]
grape rebase [-c <path>] [<git-rebase-args>...]
grape rebase [--config=<path>] [<git-rebase-args>...]
grape rebase -- <git-rebase-args>...
```

Grape removes its `--config <path>`, `--config=<path>`, and `-c <path>` options, then passes every other argument to `git rebase` in its original order. A `--` separator ends grape option handling and is removed; all following arguments go to Git, including arguments named `--config` or `-c`.

Policy matching uses the complete short current branch name, not any Git argument. During an active rebase, grape recovers the original local branch from Git's rebase state when HEAD is detached and applies the same policy. A detached HEAD outside an active rebase is rejected.

The selected config must exist and contain a policy that allows the current branch:

```json
{
  "rebase": {
    "allowed_branches": ["feature/*", "worktrees/*"]
  }
}
```

Config discovery uses explicit `--config`, then `./grape.json`, `$GRAPE_HOME/grape.json`, and `~/.grape/grape.json` when `GRAPE_HOME` is unset. Missing config exits 2. Missing `rebase`, missing `allowed_branches`, and an empty list deny every branch and exit 1 after reporting the current branch.

Patterns are case-sensitive Go `path.Match` globs over the complete branch name. `*` and `?` do not cross `/`; `**` has no recursive meaning. All patterns are validated before the current branch is inspected, so any malformed pattern exits 2 even if an earlier pattern would match.

A grape config option without a path exits 2 without invoking rebase. Grape does not validate Git rebase arguments, so Git handles unknown options, invalid combinations, and positional argument counts. A detached HEAD outside an active rebase exits 1 without invoking rebase. Git receives grape's stdin, stdout, and stderr directly, and its exit code is preserved.

Grape does not fetch, pull, infer a default branch, resolve conflicts, or roll back. Interactive mode and active-rebase operations such as `--continue`, `--abort`, `--skip`, and `--quit` pass through to Git under the same config and branch policy.
