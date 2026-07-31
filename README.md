# grape

`grape` is a Git wrapper designed for use by coding agents.

It provides agent-friendly `git worktree` workflows, including formatted worktree listing, branch lookup, prefix / regex based removal, and config-driven reset.

## Usage

```sh
grape list
```

Build a local binary with:

```sh
make build
```

## Installation

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

## Commands

| Command | Description |
| --- | --- |
| `grape list` | Show worktree path, branch, and HEAD in a colored table. |
| `grape branch <branch-name>` | Show worktrees that reference the given local branch. |
| `grape remove <prefix>` | Preview and confirm removal of worktrees under the prefix and their local branches. |
| `grape remove --regex <pattern>` | Preview and confirm removal of worktrees whose paths match the regular expression. |
| `grape init` | Create `~/.grape/grape.json` from `grape.example.json`. |
| `grape reset --config grape.json` | Recreate worktrees from config after removing non-default worktrees and local branches. |
| `grape version` | Show the build version and commit hash. |
| `grape help` | Show command help. |

Unknown commands are delegated to `git worktree`.

```sh
grape prune
grape list --porcelain
```

## Initialize Config

From a directory containing `grape.example.json`, run:

```sh
grape init
```

This creates `~/.grape/grape.json`. When `GRAPE_HOME` is set, it creates `$GRAPE_HOME/grape.json` instead. The command never overwrites an existing config file.

## Reset Config

`grape reset` reads `grape.json` by default.

`grape reset` never removes the main working tree or the branch checked out by the main working tree.
Before deleting anything, `grape reset` prints the worktrees and branches to delete and continues only after `y` or `yes` confirmation.

`grape remove` also prints its matching worktrees and branches, then continues only after `y` or `yes` confirmation.

Config path resolution:

1. `./grape.json`
2. `$GRAPE_HOME/grape.json`
3. `~/.grape/grape.json` when `GRAPE_HOME` is not set

### Migrating from gw

This is a breaking rename from `gw`. Rename `gw.json` to `grape.json`, replace `GW_HOME` with `GRAPE_HOME`, and move `~/.gw` to `~/.grape`. The old names are not resolved automatically.

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

See the [command reference](docs/references/README.md) for the full command reference.
