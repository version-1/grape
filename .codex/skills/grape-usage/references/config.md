# Configuration

Configuration-aware built-ins are `list`, `branch`, `remove`, `reset`, `push`, and `rebase`. They search for config in this order:

1. `./grape.json`
2. `$GRAPE_HOME/grape.json`
3. `~/.grape/grape.json` when `GRAPE_HOME` is unset

The first existing file is selected. The complete document is decoded strictly before command-specific work or Git inspection. Unknown or duplicate fields, case-variant field names, `null`, malformed JSON, unreadable files, invalid glob patterns, and invalid values are errors. Help, version, init, and delegated `git worktree` commands do not validate config.

When no file exists, `list`, `branch`, `remove`, `reset`, and `push` warn on stderr and use the default protected branches `main` and `master`. `rebase` instead exits 2 without inspecting Git because an existing policy file is required.

```json
{
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  }
}
```

Protected patterns use case-sensitive Go `path.Match` semantics. A configured list replaces the defaults. Omitting `push` or `protected_branches` uses the defaults. An explicit empty list is invalid. `*` does not match `/`.

Reset separately requires a non-empty, valid `worktrees` list. Other configuration-aware commands accept a push-only config.

Rebase policy uses a separate optional object:

```json
{
  "rebase": {
    "allowed_branches": ["feature/*", "worktrees/*"]
  }
}
```

Missing `rebase`, missing `allowed_branches`, and an empty list all deny rebase. Patterns use case-sensitive Go `path.Match` semantics against the complete short current branch name. Every pattern is validated before policy evaluation. `*` and `?` do not cross `/`, and `**` is not recursive.
