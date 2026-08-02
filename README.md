# grape

## Overview

`grape` is a Git wrapper designed for use by coding agents.

It provides agent-friendly `git worktree` workflows, including policy-gated rebases, safe current-branch pushes, formatted worktree listing, branch lookup, prefix / regex based removal, and config-driven reset.

## Setup

### Installation

The latest release provides binaries for macOS arm64 and Linux amd64. Download and run the installer to fetch the matching binary into the current directory:

```sh
install_script="$(mktemp)"
curl --fail --location --silent --show-error \
  https://raw.githubusercontent.com/version-1/grape/cc226a777bfb89c85130e010ad6f1e914fd8727e/scripts/install.sh \
  --output "$install_script"
sh "$install_script"
rm "$install_script"
```

The installer resolves the latest published release at runtime and refuses to overwrite an existing `./grape`. Set `GRAPE_VERSION` to a release tag such as `0.1.2` to select a specific release.

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
| `grape rebase [<git-rebase-args>...]` | Run policy-gated `git rebase` with pass-through arguments. |
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

See the [configuration reference](docs/configuration.md) for the complete `grape.json` schema, discovery rules, validation behavior, defaults, and policy semantics.

### Reset Worktrees

`grape reset` rebuilds the repository's non-default worktrees from a configuration file. It removes the existing non-default worktrees and local branches, then creates the worktrees declared in the configuration.

This provides a repeatable way to restore a known worktree layout for coding-agent sessions without manually removing and recreating each worktree.

`grape reset` is destructive, but never removes the main working tree or the branch checked out there. Before deleting anything, it prints the worktrees and branches to delete and continues only after `y` or `yes` confirmation.

`reset` requires a non-empty `worktrees` list. See the [configuration reference](docs/configuration.md) for reset fields, examples, discovery, and validation rules.

## Policy-Gated Rebase

Grape accepts Git rebase arguments while reserving `--config` and `-c` for its own config selection:

```sh
grape rebase
grape rebase origin/main
grape rebase -i --config path/to/grape.json origin/main
grape rebase --onto main base feature
grape rebase --continue
```

Grape removes `--config <path>`, `--config=<path>`, and `-c <path>` before passing every other argument to `git rebase` in its original order. A `--` separator ends grape option handling and is not passed to Git, so arguments named `--config` or `-c` can still be sent to Git after the separator.

The allow policy applies to the complete short name of the current checked-out branch, never to a positional argument. During an active rebase, when HEAD is detached, grape reads Git's rebase state and applies the policy to the original local branch.

Configure allowed current branches in `grape.json`:

```json
{
  "rebase": {
    "allowed_branches": ["feature/*", "worktrees/*"]
  }
}
```

A missing policy or empty list denies every branch. See [Allowed rebase branches](docs/configuration.md#allowed-rebase-branches) for validation and pattern semantics.

A detached HEAD outside an active rebase and denied branches exit 1 without running rebase. Missing or malformed config, invalid field types or glob patterns, and a grape config option without a path exit 2 without running rebase. Grape does not validate Git's arguments; unknown options and invalid combinations are reported by Git. Git's stdin, stdout, stderr, and exit code pass through unchanged. Conflicts are left in place and can be handled with `grape rebase --continue`, `grape rebase --abort`, or `grape rebase --skip`.

## Safe Push

Only these forms are accepted:

```sh
grape push
grape push --force-with-lease
```

The remote is always `origin`, and the destination is always the current symbolic branch. Other flags, custom lease values, refspecs, tags, remotes, and additional arguments are rejected before Git push execution. `grape` also rejects detached HEAD, non-repository execution, invalid branch names, branch names beginning with `-`, missing or ambiguous origin URLs, and protected branches. It uses exactly one `remote.origin.pushurl` when configured; otherwise it requires exactly one `remote.origin.url`.

Immediately before execution, `grape` logs the branch, selected origin URL, and exact Git command to stderr. Git's stdout and stderr pass through unchanged, and `grape` returns the exit code from `git push`.

Protected branches are configured in `grape.json`:

```json
{
  "push": {
    "protected_branches": ["main", "master", "release/*"]
  }
}
```

The defaults are `main` and `master`. See [Protected push branches](docs/configuration.md#protected-push-branches) for replacement, empty-list, discovery, and pattern semantics.

## Output and Color

Grape-generated diagnostics are written to stderr. Informational messages use `grape:`, warnings use `grape: warning:`, and errors use `grape:`. Warning and error prefixes are colored only when stderr is a terminal.

Tables, previews, prompts, and progress output determine color independently from stdout. Color is disabled for pipes and redirects and whenever `NO_COLOR` is set. Raw Git subprocess streams are not recolored.

See the [canonical command reference](.codex/skills/grape-usage/references/README.md) for complete command behavior.
