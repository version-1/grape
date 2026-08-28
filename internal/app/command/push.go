package command

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/version-1/grape/internal/logging"
)

type PushClient interface {
	CurrentBranch(context.Context) (string, error)
	ValidateBranch(context.Context, string) error
	OriginURL(context.Context) (string, error)
	Push(context.Context, string, bool, io.Writer, io.Writer) error
}
type PushCommand struct {
	Client PushClient
	Loader ConfigLoader
}

func parsePushOptions(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && args[0] == "--force-with-lease" {
		return true, nil
	}
	return false, errors.New("usage: grape push [--force-with-lease]")
}
func (c PushCommand) Run(ctx context.Context, args []string, stdout, stderr io.Writer, logger logging.Logger) int {
	cfg, code := c.Loader.Load("", logger)
	if code != 0 {
		return code
	}
	force, err := parsePushOptions(args)
	if err != nil {
		logger.Error("%v", err)
		return 2
	}
	branch, err := c.Client.CurrentBranch(ctx)
	if err != nil {
		logger.Error("current HEAD is detached or not in a git repository")
		return 1
	}
	if cfg.BranchProtected(branch) {
		logger.Error("refusing to push protected branch: %s", branch)
		return 1
	}
	if err := c.Client.ValidateBranch(ctx, branch); err != nil {
		if strings.HasPrefix(branch, "-") {
			logger.Error("refusing to push branch that starts with '-': %s", branch)
		} else {
			logger.Error("invalid branch name: %s", branch)
		}
		return 1
	}
	origin, err := c.Client.OriginURL(ctx)
	if err != nil {
		logger.Error("%v", err)
		return 1
	}
	operation := "git push origin HEAD:" + branch
	if force {
		operation = "git push --force-with-lease origin HEAD:" + branch
	}
	logger.Info("branch: %s", branch)
	logger.Info("origin: %s", origin)
	logger.Info("running: %s", operation)
	if err := c.Client.Push(ctx, branch, force, stdout, stderr); err != nil {
		if code, ok := ExitCode(err); ok {
			return code
		}
		logger.Error("%v", err)
		return 1
	}
	return 0
}
