package app

import (
	"context"
	"io"
	"os"

	"github.com/version-1/grape/internal/app/command"
	"github.com/version-1/grape/internal/color"
	"github.com/version-1/grape/internal/config"
	"github.com/version-1/grape/internal/logging"
	"github.com/version-1/grape/internal/worktree"
)

type WorktreeRunner interface {
	Run(context.Context, []string, io.Reader, io.Writer, io.Writer) error
}

type App struct {
	client            worktree.Client
	runner            WorktreeRunner
	configLoader      command.ConfigLoader
	exampleConfigPath string
	version           string
	commit            string
	colors            color.Policy
}

func (a App) WithColorPolicy(policy color.Policy) App {
	a.colors = policy
	return a
}

func New(client worktree.Client, runner WorktreeRunner, readFile config.ReadFileFunc) App {
	loader := command.ConfigLoader{
		PathResolver: config.DefaultPathResolver(),
		ReadFile:     readFile,
	}
	if readFile == nil {
		loader.ReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
		loader.PathResolver = config.PathResolver{
			Stat:     func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
			Env:      func(string) string { return "" },
			UserHome: func() (string, error) { return "/nonexistent", nil },
		}
	}
	return App{client: client, runner: runner, configLoader: loader, exampleConfigPath: "grape.example.json", version: "dev", commit: "unknown"}
}

func (a App) WithExampleConfigPath(path string) App {
	a.exampleConfigPath = path
	return a
}

func (a App) WithPathResolver(pathResolver config.PathResolver) App {
	a.configLoader.PathResolver = pathResolver
	return a
}
func (a App) WithBuildInfo(version, commit string) App {
	a.version = version
	a.commit = commit
	return a
}

type exitCodeError interface {
	ExitCode() int
}

func (a App) Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"list"}
	}
	logger := logging.New(stderr, a.colors.Stderr)
	streams := command.Streams{In: stdin, Out: stdout, Err: stderr}
	switch args[0] {
	case "help", "--help", "-h":
		return (command.HelpCommand{}).Run(stdout)
	case "version":
		return (command.VersionCommand{Version: a.version, Commit: a.commit}).Run(stdout)
	case "init":
		return (command.InitCommand{Resolver: a.configLoader.PathResolver, ExampleConfigPath: a.exampleConfigPath}).Run(args[1:], stdout, logger)
	case "list":
		if len(args) == 1 {
			return (command.ListCommand{Client: a.client, Colors: a.colors, Loader: a.configLoader}).Run(ctx, stdout, logger)
		}
	case "branch":
		return (command.BranchCommand{Client: a.client, Colors: a.colors, Loader: a.configLoader}).Run(ctx, args[1:], stdout, logger)
	case "remove":
		return (command.RemoveCommand{Client: a.client, Colors: a.colors, Loader: a.configLoader}).Run(ctx, args[1:], streams, logger)
	case "reset":
		return (command.ResetCommand{Client: a.client, Loader: a.configLoader, Colors: a.colors}).Run(ctx, args[1:], streams, logger)
	case "push":
		return (command.PushCommand{Client: a.client, Loader: a.configLoader}).Run(ctx, args[1:], stdout, stderr, logger)
	case "rebase":
		return (command.RebaseCommand{Client: a.client, Loader: a.configLoader}).Run(ctx, args[1:], streams, logger)
	}
	if err := a.runner.Run(ctx, args, stdin, stdout, stderr); err != nil {
		if exitErr, ok := err.(exitCodeError); ok {
			return exitErr.ExitCode()
		}
		logger.Error("%v", err)
		return 1
	}
	return 0
}
