package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/version-1/grape/internal/logging"
)

type RebaseClient interface {
	CurrentBranch(context.Context) (string, error)
	RebaseBranch(context.Context) (string, error)
	Rebase(context.Context, []string, io.Reader, io.Writer, io.Writer) error
}
type RebaseCommand struct {
	Client RebaseClient
	Loader ConfigLoader
}
type rebaseOptions struct {
	ConfigPath string
	GitArgs    []string
}

func parseRebaseOptions(args []string) (rebaseOptions, error) {
	options := rebaseOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			options.GitArgs = append(options.GitArgs, args[index+1:]...)
			return options, nil
		}
		if arg == "--config" || arg == "-c" {
			index++
			if index >= len(args) || args[index] == "" {
				return rebaseOptions{}, fmt.Errorf("%s requires a path", arg)
			}
			options.ConfigPath = args[index]
			continue
		}
		if strings.HasPrefix(arg, "--config=") {
			options.ConfigPath = strings.TrimPrefix(arg, "--config=")
			if options.ConfigPath == "" {
				return rebaseOptions{}, errors.New("--config requires a path")
			}
			continue
		}
		options.GitArgs = append(options.GitArgs, arg)
	}
	return options, nil
}
func (c RebaseCommand) Run(ctx context.Context, args []string, streams Streams, logger logging.Logger) int {
	options, err := parseRebaseOptions(args)
	if err != nil {
		logger.Error("usage: grape rebase [--config|-c <path>] [--] [<git-rebase-args>...]")
		return 2
	}
	cfg, code := c.Loader.LoadRebase(options.ConfigPath, logger)
	if code != 0 {
		return code
	}
	branch, err := c.Client.CurrentBranch(ctx)
	if err != nil {
		branch, err = c.Client.RebaseBranch(ctx)
		if err != nil {
			logger.Error("current HEAD is detached outside an active rebase or not in a git repository")
			return 1
		}
	}
	if !cfg.BranchAllowedForRebase(branch) {
		logger.Error("rebase is not allowed for current branch: %s", branch)
		return 1
	}
	if err := c.Client.Rebase(ctx, options.GitArgs, streams.In, streams.Out, streams.Err); err != nil {
		if code, ok := ExitCode(err); ok {
			return code
		}
		logger.Error("%v", err)
		return 1
	}
	return 0
}
