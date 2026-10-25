package command

import (
	"context"
	"io"
	"strings"

	"github.com/version-1/grape/internal/config"
	"github.com/version-1/grape/internal/logging"
)

type StartClient interface {
	RemoteDefaultBranch(context.Context) (string, error)
	ValidateBranch(context.Context, string) error
	StartBranch(context.Context, string, string, io.Writer, io.Writer) error
}

type StartCommand struct {
	Client StartClient
	Loader ConfigLoader
	Prefix string
}

func (c StartCommand) Run(ctx context.Context, args []string, stdout, stderr io.Writer, logger logging.Logger) int {
	cfg, code := c.Loader.Load("", logger)
	if code != 0 {
		return code
	}
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		logger.Error("usage: grape start-%s <branch-name>", c.Prefix)
		return 2
	}
	branch := c.Prefix + "/" + args[0]
	if cfg.BranchProtected(branch) {
		logger.Error("refusing to create protected branch: %s", branch)
		return 1
	}
	if err := c.Client.ValidateBranch(ctx, branch); err != nil {
		logger.Error("invalid branch name: %s", branch)
		return 1
	}
	defaultBranch, err := startDefaultBranch(ctx, c.Client, cfg)
	if err != nil {
		logger.Error("determine default branch: %v", err)
		if code, ok := ExitCode(err); ok {
			return code
		}
		return 1
	}
	if err := c.Client.StartBranch(ctx, defaultBranch, branch, stdout, stderr); err != nil {
		logger.Error("start branch: %v", err)
		if code, ok := ExitCode(err); ok {
			return code
		}
		return 1
	}
	return 0
}

func startDefaultBranch(ctx context.Context, client StartClient, cfg config.Config) (string, error) {
	if cfg.DefaultBranch != "" {
		return cfg.DefaultBranch, nil
	}
	return client.RemoteDefaultBranch(ctx)
}
