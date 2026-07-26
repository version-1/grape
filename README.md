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

The `0.1.0` release provides binaries for macOS arm64 and Linux amd64. Download and run the installer to fetch the matching binary into the current directory:

```sh
install_script="$(mktemp)"
curl --fail --location --silent --show-error \
  https://raw.githubusercontent.com/version-1/grape/15d9f8ca7f96ebfd16a187879a81167d0b1ea073/scripts/install.sh \
  --output "$install_script"
sh "$install_script"
rm "$install_script"
```

The installer refuses to overwrite an existing `./grape`. Set `GW_VERSION` before running it to select a different release.

Install the downloaded binary on your `PATH` manually:

```sh
chmod +x ./grape
mkdir -p "$HOME/.local/bin"
mv ./grape "$HOME/.local/bin/grape"
```

Add `~/.local/bin` to your zsh `PATH` persistently:

```sh
if ! grep -qxF 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zshrc"; then
  echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
fi
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
