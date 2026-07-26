# gw

`gw` is a Go CLI for `git worktree` workflows.

It wraps `git worktree` and adds formatted worktree listing, branch lookup, prefix / regex based removal, and config-driven reset.

## Usage

```sh
cd shared/commands/gw
go run ./cmd/gw list
```

Build a local binary with:

```sh
make build
```

## Installation

The `0.1.0` release provides a macOS arm64 binary. Install it under `~/.local/bin` with the GitHub CLI:

```sh
mkdir -p "$HOME/.local/bin"
gh release download 0.1.0 --repo version-1/grape --pattern gw_0.1.0_darwin_arm64 --dir "$HOME/.local/bin"
mv "$HOME/.local/bin/gw_0.1.0_darwin_arm64" "$HOME/.local/bin/gw"
chmod +x "$HOME/.local/bin/gw"
```

Ensure `~/.local/bin` is on your `PATH`. For zsh:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

## Commands

| Command | Description |
| --- | --- |
| `gw list` | Show worktree path, branch, and HEAD in a colored table. |
| `gw branch <branch-name>` | Show worktrees that reference the given local branch. |
| `gw remove <prefix>` | Preview and confirm removal of worktrees under the prefix and their local branches. |
| `gw remove --regex <pattern>` | Preview and confirm removal of worktrees whose paths match the regular expression. |
| `gw reset --config gw.json` | Recreate worktrees from config after removing non-default worktrees and local branches. |
| `gw help` | Show command help. |

Unknown commands are delegated to `git worktree`.

```sh
go run ./cmd/gw prune
go run ./cmd/gw list --porcelain
```

## Reset Config

`gw reset` reads `gw.json` by default.

`gw reset` never removes the main working tree or the branch checked out by the main working tree.
Before deleting anything, `gw reset` prints the worktrees and branches to delete and continues only after `y` or `yes` confirmation.

`gw remove` also prints its matching worktrees and branches, then continues only after `y` or `yes` confirmation.

Config path resolution:

1. `./gw.json`
2. `$GW_HOME/gw.json`
3. `~/.gw/gw.json` when `GW_HOME` is not set

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

See [REFERENCE.md](REFERENCE.md) for the full command reference.
