# grape

`grape` is a Go CLI for `git worktree` workflows.

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
  https://raw.githubusercontent.com/version-1/grape/aa71316a5da0dd1bf442c6ec62e51e51bbf28d57/scripts/install.sh \
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
| `grape list` | Show worktree path, branch, and HEAD in a colored table. |
| `grape branch <branch-name>` | Show worktrees that reference the given local branch. |
| `grape remove <prefix>` | Preview and confirm removal of worktrees under the prefix and their local branches. |
| `grape remove --regex <pattern>` | Preview and confirm removal of worktrees whose paths match the regular expression. |
| `grape reset --config gw.json` | Recreate worktrees from config after removing non-default worktrees and local branches. |
| `grape help` | Show command help. |

Unknown commands are delegated to `git worktree`.

```sh
go run ./cmd/gw prune
go run ./cmd/gw list --porcelain
```

## Reset Config

`grape reset` reads `gw.json` by default.

`grape reset` never removes the main working tree or the branch checked out by the main working tree.
Before deleting anything, `grape reset` prints the worktrees and branches to delete and continues only after `y` or `yes` confirmation.

`grape remove` also prints its matching worktrees and branches, then continues only after `y` or `yes` confirmation.

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
