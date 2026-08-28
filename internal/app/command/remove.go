package command

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/version-1/grape/internal/color"
	"github.com/version-1/grape/internal/logging"
	"github.com/version-1/grape/internal/ui"
	"github.com/version-1/grape/internal/worktree"
)

type RemoveClient interface {
	ListWorktrees(context.Context) ([]worktree.Worktree, error)
	RemoveWorktree(context.Context, string, bool, io.Writer, io.Writer) error
	DeleteBranch(context.Context, string, io.Writer, io.Writer) error
}
type RemoveCommand struct {
	Client RemoveClient
	Colors color.Policy
	Loader ConfigLoader
}
type removeOptions struct {
	Pattern string
	Regex   bool
}

func parseRemoveOptions(args []string) (removeOptions, error) {
	flags := flag.NewFlagSet("remove", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	regex := flags.Bool("regex", false, "match worktree paths by regular expression")
	flags.BoolVar(regex, "r", false, "match worktree paths by regular expression")
	if err := flags.Parse(args); err != nil {
		return removeOptions{}, err
	}
	if flags.NArg() != 1 {
		return removeOptions{}, errors.New("usage: grape remove [--regex|-r] <path-prefix-or-pattern>")
	}
	return removeOptions{Pattern: flags.Arg(0), Regex: *regex}, nil
}
func (c RemoveCommand) Run(ctx context.Context, args []string, streams Streams, logger logging.Logger) int {
	if _, code := c.Loader.Load("", logger); code != 0 {
		return code
	}
	options, err := parseRemoveOptions(args)
	if err != nil {
		logger.Error("%v", err)
		return 2
	}
	matcher, err := worktree.NewMatcher(worktree.MatchOptions{Pattern: options.Pattern, Regex: options.Regex})
	if err != nil {
		logger.Error("%v", err)
		return 2
	}
	items, err := c.Client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("list worktrees: %v", err)
		return 1
	}
	targets := worktree.Filter(items, func(item worktree.Worktree) bool { return !item.Main && matcher(item) })
	if len(targets) == 0 {
		fmt.Fprintln(streams.Out, ui.Paint(c.Colors.Stdout, "grape: no matching worktrees", ui.Dim))
		return 0
	}
	if !confirmRemove(streams.In, streams.Out, targets, c.Colors.Stdout, logger) {
		return 1
	}
	for _, target := range targets {
		if code := c.removeWorktreeAndBranch(ctx, target, streams); code != 0 {
			return code
		}
	}
	return 0
}
func confirmRemove(stdin io.Reader, stdout io.Writer, targets []worktree.Worktree, colorEnabled bool, logger logging.Logger) bool {
	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "grape remove will delete:", ui.Red, ui.Bold))
	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "Worktrees", ui.Blue, ui.Bold))
	fmt.Fprint(stdout, ui.FormatWorktreeList(targets, colorEnabled))
	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "Branches", ui.Red, ui.Bold))
	for _, target := range targets {
		if target.Branch != "" {
			fmt.Fprintf(stdout, "  %s\n", ui.Paint(colorEnabled, target.Branch, ui.Green))
		}
	}
	fmt.Fprint(stdout, ui.Paint(colorEnabled, "Proceed with removal? [y/N] ", ui.Red, ui.Bold))
	if stdin == nil {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, ui.Paint(colorEnabled, "grape: removal cancelled", ui.Dim))
		return false
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		logger.Error("read confirmation: %v", err)
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "y" || answer == "yes" {
		return true
	}
	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "grape: removal cancelled", ui.Dim))
	return false
}
func (c RemoveCommand) removeWorktreeAndBranch(ctx context.Context, item worktree.Worktree, streams Streams) int {
	fmt.Fprintf(streams.Out, "%s %s\n", ui.Paint(c.Colors.Stdout, "Removing worktree", ui.Blue, ui.Bold), ui.Paint(c.Colors.Stdout, item.Path, ui.Cyan))
	if err := c.Client.RemoveWorktree(ctx, item.Path, false, streams.Out, streams.Err); err != nil {
		logging.New(streams.Err, c.Colors.Stderr).Error("remove worktree %s: %v", item.Path, err)
		return 1
	}
	if item.Branch == "" {
		return 0
	}
	fmt.Fprintf(streams.Out, "%s %s\n", ui.Paint(c.Colors.Stdout, "Deleting branch", ui.Red, ui.Bold), ui.Paint(c.Colors.Stdout, item.Branch, ui.Green))
	if err := c.Client.DeleteBranch(ctx, item.Branch, streams.Out, streams.Err); err != nil {
		logging.New(streams.Err, c.Colors.Stderr).Error("delete branch %s: %v", item.Branch, err)
		return 1
	}
	return 0
}
