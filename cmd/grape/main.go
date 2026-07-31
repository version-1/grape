package main

import (
	"context"
	"os"

	"github.com/version-1/grape/internal/app"
	"github.com/version-1/grape/internal/color"
	"github.com/version-1/grape/internal/worktree"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	client := worktree.CommandClient{}
	application := app.New(client, client, os.ReadFile).
		WithBuildInfo(version, commit).
		WithColorPolicy(color.Detect(os.Stdout, os.Stderr, os.Getenv))
	os.Exit(application.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
