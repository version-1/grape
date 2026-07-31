# grape Command Reference

`grape` is a Git wrapper for safe, agent-friendly worktree workflows.

The canonical, task-oriented reference is bundled with the [`grape-usage` skill](.codex/skills/grape-usage/references/README.md). This file is retained as a standalone compatibility reference.

## Commands

### `grape` and `grape list`

List worktrees in a formatted table. A configuration file, when present, must pass strict whole-document validation before Git is called.

### `grape branch <branch-name>`

List worktrees that reference the normalized local branch name.

### `grape remove [--regex|-r] <path-prefix-or-pattern>`

Preview matching non-main worktrees and their local branches, then require `y` or `yes` before deletion. The main working tree is always excluded.

### `grape reset [--config|-c <path>]`

Strictly decode the selected config, require a non-empty `worktrees` list, preview deletion targets, and require confirmation. Reset never removes the main working tree, its checked-out branch, or the resolved default branch.

### `grape push`

After config validation and safety checks, execute:

```text
git push origin HEAD:<current-branch>
```

### `grape push --force-with-lease`

After the same validation and safety checks, execute:

```text
git push --force-with-lease origin HEAD:<current-branch>
```

No other `push` arguments are accepted. In particular, `--force`, `-f`, custom lease values, tags, refspecs, custom remotes, multiple branches, and additional arguments are rejected. Push never prompts.

Push requires:

- a symbolic current branch in a Git repository;
- a branch name that does not begin with `-` and passes `git check-ref-format --branch`;
- a branch that does not match a protected pattern;
- exactly one `remote.origin.pushurl`, or, when no push URL exists, exactly one `remote.origin.url`.

Push URL takes precedence over fetch URL. Missing or multiple selected URLs are errors. Before Git runs, the branch, selected origin URL, and exact command are logged to stderr. Git subprocess streams pass through unchanged, and the `git push` exit code is preserved.

### `grape init`

Create `grape.json` under `$GRAPE_HOME` or `~/.grape` from `grape.example.json`. Existing files are never overwritten.

### `grape version`

Print the build version and commit.

### `grape help`

Print built-in help.

### Unknown commands

Unknown commands, including non-built-in forms such as `grape list --porcelain`, are delegated to `git worktree` without grape config validation.

## Configuration

Search order when `--config` is not available or omitted:

1. `./grape.json`
2. `$GRAPE_HOME/grape.json`
3. `~/.grape/grape.json` when `GRAPE_HOME` is unset

The first existing file is selected. An unreadable, malformed, or invalid selected file is an error; grape does not fall through to a later file. If no file exists, configuration-aware commands warn and use the default protected patterns `main` and `master`.

Example:

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
    }
  ]
}
```

The complete document is decoded strictly. Unknown fields are rejected at every level. Common structural validation is applied to `list`, `branch`, `remove`, `reset`, and `push`; reset additionally requires valid worktree entries.

`push.protected_branches` uses case-sensitive Go `path.Match` semantics. An explicit list replaces the defaults. An omitted `push` section or omitted `protected_branches` field uses the defaults. An explicit empty list, empty pattern, or invalid glob is rejected. `*` does not match `/`.

## Logging and Color

Grape-generated stderr formats are:

```text
grape: <information>
grape: warning: <warning>
grape: <error>
```

Only warning and error prefixes are colored, and only when stderr is a terminal. Stdout UI output uses an independent terminal check. Both streams disable color when redirected or piped, and `NO_COLOR` disables all grape color. Help, version, stdout results, prompts, previews, progress, and raw Git streams remain outside the stderr logger.
