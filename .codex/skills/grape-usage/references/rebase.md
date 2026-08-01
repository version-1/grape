# `grape rebase`

Rebases the current symbolic branch only when a strict config policy allows that branch.

Accepted forms:

```sh
grape rebase
grape rebase <upstream>
grape rebase --config <path> [<upstream>]
grape rebase -c <path> [<upstream>]
```

Without an upstream, grape executes `git rebase`, so Git uses the current branch's configured upstream. With one upstream, it executes `git rebase <upstream>`. Policy matching always uses the complete short current branch name, not the upstream argument.

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

Unknown options, Git rebase flags, and more than one upstream exit 2 without invoking rebase. Detached HEAD exits 1 without invoking rebase. Git receives grape's stdin, stdout, and stderr directly, and its exit code is preserved.

Grape does not fetch, pull, infer a default branch, resolve conflicts, or roll back. It does not support interactive mode, `--continue`, `--abort`, or `--skip`; invoke `git rebase` directly for those operations.
