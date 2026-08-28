package command

import (
	"fmt"
	"io"
	"strings"
)

type HelpCommand struct{}

func (HelpCommand) Run(stdout io.Writer) int { fmt.Fprint(stdout, helpText()); return 0 }

func helpText() string {
	return strings.TrimLeft(`
grape is a Go CLI for git worktree workflows.

Usage:
  grape
  grape list
  grape branch <branch-name>
  grape remove [--regex|-r] <path-prefix-or-pattern>
  grape reset [--yes|-y] [--config|-c <path>]
  grape push [--force-with-lease]
  grape rebase [--config|-c <path>] [--] [<git-rebase-args>...]
  grape init
  grape version
  grape help

Commands:
  list      Show worktree path, branch, and HEAD in a colored table.
  branch    Show worktrees that reference the given local branch.
  remove    Remove matching worktrees and their local branches.
  reset     Recreate worktrees from config after removing non-default worktrees and local branches.
  push      Safely push the current branch to origin.
  rebase    Rebase the current branch when allowed by config.
  init      Create grape.json in the resolved config home from grape.example.json.
  version   Show the build version and commit hash.
  help      Show this help.

Config:
  Configuration-aware commands read config in this order:
    1. ./grape.json
    2. $GRAPE_HOME/grape.json
    3. ~/.grape/grape.json
  Config is decoded strictly. Missing config uses protected defaults main and master,
  except rebase, which requires an existing config file.

Safety:
  push accepts no arguments or exactly --force-with-lease, always targets origin,
  and refuses protected branches.
  rebase validates its complete allow policy before running git rebase.
  Except for grape's config option, rebase arguments pass through to git rebase.
  reset and remove never remove the main working tree.
  reset also keeps the branch checked out by the main working tree.
  reset shows deletion targets and continues only after y/yes confirmation unless --yes/-y is used.

Unknown commands are delegated to git worktree.

Color is automatic per output stream and disabled by NO_COLOR.
`, "\n")
}
