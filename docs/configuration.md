# Configuration Reference

`grape.json` is the configuration file for grape's worktree reset, protected push branch, and allowed rebase branch policies. This page is the canonical reference for the complete configuration contract.

## Complete Structure

All fields are optional at the document level, although individual commands can require fields. A combined configuration can contain every supported field:

```json
{
  "default_branch": "main",
  "worktrees": [
    {
      "path": ".worktrees/api",
      "branch": "worktrees/api",
      "start_point": "origin/main"
    }
  ],
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  },
  "rebase": {
    "allowed_branches": ["feature/*", "worktrees/*"]
  }
}
```

The top-level value must be a JSON object. `null` is not accepted for the document, an object, a field, or an array element.

## Field Reference

| JSON path | Type | Required | Default | Consuming commands |
| --- | --- | --- | --- | --- |
| `default_branch` | string | No | Detected from `refs/remotes/origin/HEAD` by `reset` | `reset` |
| `worktrees` | array of worktree objects | Required and non-empty for `reset`; otherwise optional | None | `reset` |
| `worktrees[].path` | string | Yes for each worktree; must be non-empty | None | `reset` |
| `worktrees[].branch` | string | Yes for each worktree; must be non-empty | None | `reset` |
| `worktrees[].start_point` | string | No | The resolved `default_branch` | `reset` |
| `push` | object | No | Built-in push policy | `push` |
| `push.protected_branches` | array of strings | No; an explicit array must be non-empty | `["main", "master"]` | `push` |
| `rebase` | object | No | Deny all branches | `rebase` |
| `rebase.allowed_branches` | array of strings | No | Empty policy, which denies all branches | `rebase` |

Every configuration-aware command validates the whole document, including fields it does not operationally consume. For example, an invalid rebase pattern also prevents `grape list` and `grape push` from running.

### Reset and worktree fields

`default_branch` identifies the branch that `reset` preserves and uses as the default start point. If it is omitted or is an empty string, grape reads `refs/remotes/origin/HEAD` and removes an `origin/` prefix from the result. A `refs/heads/` prefix in a configured value is normalized away before use. If no non-empty default branch can be resolved, `reset` fails.

`worktrees` must contain at least one item for `reset`. Other configuration-aware commands accept an omitted or empty `worktrees` array.

Each `worktrees[].path` must be non-empty and resolve to a unique absolute path. Relative paths are resolved from grape's current working directory, not from the configuration file's directory. Lexically equivalent paths, such as `worktree`, `./worktree`, and the corresponding absolute path, are duplicates.

Each `worktrees[].branch` must be non-empty and unique after an optional `refs/heads/` prefix is removed. The configuration validator does not otherwise validate Git branch syntax; Git can still reject a branch when grape creates the worktree.

`worktrees[].start_point` is passed to Git as the start point. When it is omitted or is an empty string, grape uses the resolved `default_branch`. Grape creates each entry with:

```text
git worktree add -B <branch> <path> <start_point>
```

Before creation, `reset` previews and asks to delete non-default worktrees and local branches. It never removes the main working tree or the branch checked out there.

### Protected push branches

`push.protected_branches` contains branch patterns that `grape push` refuses to push. Omitting either `push` or `protected_branches` uses the built-in patterns `main` and `master`.

An explicitly configured list replaces the built-in patterns; it does not extend them. Include `main` and `master` yourself if they must remain protected. An explicit empty list is invalid, so configuration cannot disable all push protection. Every entry must be a non-empty, syntactically valid branch pattern.

The policy is checked against the complete short name of the current symbolic branch. It applies to both `grape push` and `grape push --force-with-lease`.

### Allowed rebase branches

`rebase.allowed_branches` contains branch patterns on which `grape rebase` may run. A missing `rebase` object, a missing `allowed_branches` field, or an explicit empty list denies all branches. Unlike push protection, rebase has no built-in allow patterns.

All patterns are validated before grape inspects the current branch. An empty string is a syntactically valid pattern but cannot match a non-empty branch name, so it grants no permission.

The policy is checked against the complete short name of the current symbolic branch, not the optional rebase upstream. It applies to both `grape rebase` and `grape rebase <upstream>`.

## Branch Pattern Semantics

Protected branches and allowed branches use Go [`path.Match`](https://pkg.go.dev/path#Match) syntax. Matching is case-sensitive and must match the complete branch name.

- `*` matches any sequence of non-`/` characters.
- `?` matches one non-`/` character.
- Character classes such as `[0-9]` use `path.Match` syntax.
- `/` is a separator and is matched only by `/`.
- `**` has no recursive special meaning; it behaves as adjacent `*` patterns and still does not cross `/`.

Examples:

| Pattern | Branch | Matches |
| --- | --- | --- |
| `release/*` | `release/1.0` | Yes |
| `release/*` | `release/series/1.0` | No |
| `feature/?` | `feature/a` | Yes |
| `feature/?` | `feature/api` | No |
| `main` | `Main` | No |
| `feature/**` | `feature/team/api` | No |

## Discovery and Precedence

`reset` and `rebase` accept `--config <path>` and `-c <path>`. The explicit path is used exactly as provided and has highest precedence. The other configuration-aware commands do not accept an explicit configuration option.

Without an explicit path, grape selects the first existing file in this order:

1. `./grape.json` in the current working directory.
2. `$GRAPE_HOME/grape.json` when `GRAPE_HOME` is non-empty.
3. `~/.grape/grape.json` when `GRAPE_HOME` is unset or empty.

When `GRAPE_HOME` is set, grape does not subsequently try `~/.grape/grape.json`. The current-directory file always takes precedence over the user-level file.

`grape init` does not use the search order. It creates `$GRAPE_HOME/grape.json`, or `~/.grape/grape.json` when `GRAPE_HOME` is unset, from the bundled example and refuses to overwrite an existing file.

## Validation and Error Behavior

The configuration-aware built-in commands are `list`, `branch`, `remove`, `reset`, `push`, and `rebase`. They decode and validate the complete selected document before inspecting Git. `help`, `version`, `init`, and commands delegated to `git worktree` do not read or validate the configuration file.

Validation is strict:

- Unknown top-level or nested fields are errors. Field names are case-sensitive, so a case variant is unknown.
- Duplicate object fields are errors.
- `null`, malformed JSON, multiple JSON values, trailing non-whitespace content, and incorrect JSON types are errors.
- Invalid branch pattern syntax anywhere in the document is an error.
- A selected file that cannot be read is an error; grape does not fall through to a lower-precedence file.
- A filesystem error while checking a candidate path is an error; only a not-found result continues discovery.

An explicit path is considered selected even when it does not exist, so a missing explicit file is a read error.

When no discovered file exists, `list`, `branch`, `remove`, and `push` continue with the built-in push protection and print this warning to stderr:

```text
grape: warning: grape.json not found; using default protected branches: main, master
```

`reset` prints the same warning, then fails command-specific validation because the default in-memory configuration has no `worktrees`. `rebase` instead reports that the config was not found and exits before Git inspection because rebase requires an existing allow policy.

Global decoding and validation errors exit with status 2 before Git inspection. Command policy denials, such as a protected push branch or a current branch not allowed to rebase, exit with status 1.

## Examples

### Reset and worktree layout

```json
{
  "default_branch": "main",
  "worktrees": [
    {
      "path": ".worktrees/frontend",
      "branch": "worktrees/frontend",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/backend",
      "branch": "worktrees/backend"
    }
  ]
}
```

### Protected push branches

This replacement policy preserves the built-in protections and adds release branches:

```json
{
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  }
}
```

### Allowed rebase branches

This allow policy permits one-level feature and worktree branch names. Nested names such as `feature/team/api` remain denied.

```json
{
  "rebase": {
    "allowed_branches": ["feature/*", "worktrees/*"]
  }
}
```

### Combined configuration

```json
{
  "default_branch": "main",
  "worktrees": [
    {
      "path": ".worktrees/agent-1",
      "branch": "worktrees/agent-1",
      "start_point": "origin/main"
    }
  ],
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  },
  "rebase": {
    "allowed_branches": ["feature/*", "worktrees/*"]
  }
}
```
