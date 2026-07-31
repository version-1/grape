# grape

## Overview

`grape` is a Git wrapper designed for use by coding agents.

It provides agent-friendly `git worktree` workflows, including safe current-branch pushes, formatted worktree listing, branch lookup, prefix / regex based removal, and config-driven reset.

## Setup

### Installation

The `0.1.2` release provides binaries for macOS arm64 and Linux amd64. Download and run the installer to fetch the matching binary into the current directory:

```sh
install_script="$(mktemp)"
curl --fail --location --silent --show-error \
  https://raw.githubusercontent.com/version-1/grape/cdff260d5078c7f0db67e2ab59ddaa552d5798d7/scripts/install.sh \
  --output "$install_script"
sh "$install_script"
rm "$install_script"
```

The installer refuses to overwrite an existing `./grape`. Set `GRAPE_VERSION` before running it to select a different release.

Install the downloaded binary on your `PATH` manually:

```sh
chmod +x ./grape
mkdir -p "$HOME/.local/bin"
if [ -e "$HOME/.local/bin/grape" ] || [ -L "$HOME/.local/bin/grape" ]; then
  echo "Refusing to overwrite $HOME/.local/bin/grape" >&2
  exit 1
fi
mv ./grape "$HOME/.local/bin/grape"
```

Add `~/.local/bin` to your zsh `PATH` persistently:

```sh
if ! grep -qxF 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zshrc"; then
  echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
fi
export PATH="$HOME/.local/bin:$PATH"
```

### Initialize Config

From a directory containing `grape.example.json`, run:

```sh
grape init
```

This creates `~/.grape/grape.json`. When `GRAPE_HOME` is set, it creates `$GRAPE_HOME/grape.json` instead. The command never overwrites an existing config file.

## Usage

```sh
grape list
```

Build a local binary with:

```sh
make build
```

### Commands

| Command | Description |
| --- | --- |
| `grape list` | Show worktree path, branch, and HEAD in a colored table. |
| `grape branch <branch-name>` | Show worktrees that reference the given local branch. |
| `grape remove <prefix>` | Preview and confirm removal of worktrees under the prefix and their local branches. |
| `grape remove --regex <pattern>` | Preview and confirm removal of worktrees whose paths match the regular expression. |
| `grape push` | Push the current branch with `git push origin HEAD:<current-branch>`. |
| `grape push --force-with-lease` | Push the current branch with Git's exact `--force-with-lease` option. |
| `grape init` | Create `~/.grape/grape.json` from `grape.example.json`. |
| `grape reset --config grape.json` | Recreate worktrees from config after removing non-default worktrees and local branches. |
| `grape version` | Show the build version and commit hash. |
| `grape help` | Show command help. |

Unknown commands are delegated to `git worktree`.

```sh
grape prune
grape list --porcelain
```

See the [command reference](.codex/skills/grape-usage/references/README.md) for all commands, options, and safety behavior.

### Reset Worktrees

`grape reset` rebuilds the repository's non-default worktrees from a configuration file. It removes the existing non-default worktrees and local branches, then creates the worktrees declared in the configuration.

This provides a repeatable way to restore a known worktree layout for coding-agent sessions without manually removing and recreating each worktree.

`grape reset` is destructive, but never removes the main working tree or the branch checked out there. Before deleting anything, it prints the worktrees and branches to delete and continues only after `y` or `yes` confirmation.

By default, `grape reset` resolves `grape.json` in this order:

1. `./grape.json`
2. `$GRAPE_HOME/grape.json`
3. `~/.grape/grape.json` when `GRAPE_HOME` is not set

The complete JSON document is decoded strictly for `list`, `branch`, `remove`, `reset`, and `push`. Unknown top-level fields, unknown nested fields, malformed JSON, unreadable selected files, and invalid values stop the command before Git is inspected. `help`, `version`, `init`, and commands delegated to `git worktree` do not validate config.

`reset` requires a non-empty `worktrees` list. Other built-in commands may use a push-only config without `worktrees`.

## Safe Push

Only these forms are accepted:

```sh
grape push
grape push --force-with-lease
```

The remote is always `origin`, and the destination is always the current symbolic branch. Other flags, custom lease values, refspecs, tags, remotes, and additional arguments are rejected before Git push execution. `grape` also rejects detached HEAD, non-repository execution, invalid branch names, branch names beginning with `-`, missing or ambiguous origin URLs, and protected branches. It uses exactly one `remote.origin.pushurl` when configured; otherwise it requires exactly one `remote.origin.url`.

Immediately before execution, `grape` logs the branch, selected origin URL, and exact Git command to stderr. Git's stdout and stderr pass through unchanged, and `grape` returns the exit code from `git push`.

Protected branches are configured with case-sensitive Go `path.Match` patterns:

```json
{
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  }
}
```

The defaults are `main` and `master`. An explicitly configured list replaces the defaults; it does not extend them. Omitting `push` or `protected_branches` uses the defaults. An explicit empty list and invalid glob patterns are configuration errors. Because `path.Match` treats `/` as a separator, `release/*` matches `release/1.0` but not `release/series/1.0`.

When no config file exists, configuration-aware commands continue with the defaults and write this warning to stderr:

```text
grape: warning: grape.json not found; using default protected branches: main, master
```

## Output and Color

Grape-generated diagnostics are written to stderr. Informational messages use `grape:`, warnings use `grape: warning:`, and errors use `grape:`. Warning and error prefixes are colored only when stderr is a terminal.

Tables, previews, prompts, and progress output determine color independently from stdout. Color is disabled for pipes and redirects and whenever `NO_COLOR` is set. Raw Git subprocess streams are not recolored.
```json
{
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
    },
    {
      "path": ".worktrees/3",
      "branch": "worktrees/3",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/4",
      "branch": "worktrees/4",
      "start_point": "origin/main"
    },
    {
      "path": ".worktrees/5",
      "branch": "worktrees/5",
      "start_point": "origin/main"
    }
  ]
}
```

`reset` is destructive. It removes worktrees and local branches except the default branch before creating configured worktrees.

See the [command reference](REFERENCE.md) for the full command reference.
