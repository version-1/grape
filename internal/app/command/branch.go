package command

import (
	"context"
	"fmt"
	"io"

	"github.com/version-1/grape/internal/color"
	"github.com/version-1/grape/internal/logging"
	"github.com/version-1/grape/internal/ui"
	"github.com/version-1/grape/internal/worktree"
)

type BranchClient interface {
	ListWorktrees(context.Context) ([]worktree.Worktree, error)
}
type BranchCommand struct {
	Client BranchClient
	Colors color.Policy
	Loader ConfigLoader
}

func (c BranchCommand) Run(ctx context.Context, args []string, stdout io.Writer, logger logging.Logger) int {
	if _, code := c.Loader.Load("", logger); code != 0 {
		return code
	}
	if len(args) != 1 {
		logger.Error("usage: grape branch <branch-name>")
		return 2
	}
	branch := worktree.NormalizeBranchName(args[0])
	items, err := c.Client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("%v", err)
		return 1
	}
	targets := worktree.Filter(items, func(item worktree.Worktree) bool { return item.Branch == branch })
	if len(targets) == 0 {
		fmt.Fprintf(stdout, "%s %s\n", ui.Paint(c.Colors.Stdout, "No worktree for branch", ui.Dim), ui.Paint(c.Colors.Stdout, branch, ui.Green))
		return 1
	}
	fmt.Fprint(stdout, ui.FormatWorktreeList(targets, c.Colors.Stdout))
	return 0
}
