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

type ListClient interface {
	ListWorktrees(context.Context) ([]worktree.Worktree, error)
}
type ListCommand struct {
	Client ListClient
	Colors color.Policy
	Loader ConfigLoader
}

func (c ListCommand) Run(ctx context.Context, stdout io.Writer, logger logging.Logger) int {
	if _, code := c.Loader.Load("", logger); code != 0 {
		return code
	}
	items, err := c.Client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("%v", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout, ui.Paint(c.Colors.Stdout, "No worktrees", ui.Dim))
		return 0
	}
	fmt.Fprint(stdout, ui.FormatWorktreeList(items, c.Colors.Stdout))
	return 0
}
