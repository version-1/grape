package main

import (
	"context"
	"os"

	"github.com/version-1/dotfiles/shared/commands/gw/internal/app"
	"github.com/version-1/dotfiles/shared/commands/gw/internal/worktree"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	client := worktree.CommandClient{}
	application := app.New(client, client, os.ReadFile).WithBuildInfo(version, commit)
	os.Exit(application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
