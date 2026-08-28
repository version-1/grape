package command

import (
	"bufio"
	"bytes"
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

const resetRemoveConcurrency = 5
const resetAddConcurrency = 1

type ResetClient interface {
	DefaultBranch(context.Context) (string, error)
	ListWorktrees(context.Context) ([]worktree.Worktree, error)
	ListBranches(context.Context) ([]string, error)
	RemoveWorktree(context.Context, string, bool, io.Writer, io.Writer) error
	DeleteBranches(context.Context, []string, io.Writer, io.Writer) error
	AddWorktree(context.Context, worktree.ConfiguredItem, string, io.Writer, io.Writer) error
	ListRawBranches(context.Context, io.Writer, io.Writer) error
}
type ResetCommand struct {
	Client ResetClient
	Loader ConfigLoader
	Colors color.Policy
}
type resetOptions struct {
	ConfigPath string
	Yes        bool
}

func parseResetOptions(args []string) (resetOptions, error) {
	flags := flag.NewFlagSet("reset", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("config", "", "path to grape reset config")
	flags.StringVar(path, "c", "", "path to grape reset config")
	yes := flags.Bool("yes", false, "skip reset confirmation")
	flags.BoolVar(yes, "y", false, "skip reset confirmation")
	if err := flags.Parse(args); err != nil {
		return resetOptions{}, err
	}
	if flags.NArg() != 0 {
		return resetOptions{}, errors.New("usage: grape reset [--yes|-y] [--config|-c <path>]")
	}
	return resetOptions{ConfigPath: *path, Yes: *yes}, nil
}
func (c ResetCommand) Run(ctx context.Context, args []string, streams Streams, logger logging.Logger) int {
	options, err := parseResetOptions(args)
	if err != nil {
		logger.Error("%v", err)
		return 2
	}
	cfg, code := c.Loader.Load(options.ConfigPath, logger)
	if code != 0 {
		return code
	}
	if err := cfg.ValidateReset(); err != nil {
		logger.Error("read config: %v", err)
		return 2
	}
	defaultBranch := cfg.DefaultBranch
	if defaultBranch == "" {
		defaultBranch, err = c.Client.DefaultBranch(ctx)
		if err != nil {
			logger.Error("detect default branch: %v", err)
			return 1
		}
	}
	defaultBranch = worktree.NormalizeBranchName(defaultBranch)
	if defaultBranch == "" {
		logger.Error("default branch is empty")
		return 2
	}
	items, err := c.Client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("list worktrees: %v", err)
		return 1
	}
	protected := map[string]struct{}{defaultBranch: {}}
	removals := make([]worktree.Worktree, 0)
	for _, item := range items {
		if item.Main || item.Branch == defaultBranch {
			if item.Branch != "" {
				protected[item.Branch] = struct{}{}
			}
			continue
		}
		removals = append(removals, item)
	}
	branches, err := c.Client.ListBranches(ctx)
	if err != nil {
		logger.Error("list branches: %v", err)
		return 1
	}
	deletes := make([]string, 0)
	for _, branch := range branches {
		branch = worktree.NormalizeBranchName(branch)
		if branch == "" {
			continue
		}
		if _, ok := protected[branch]; !ok {
			deletes = append(deletes, branch)
		}
	}
	showResetTargets(streams.Out, removals, deletes, c.Colors.Stdout)
	showResetCounts(streams.Out, len(removals), len(cfg.Worktrees), c.Colors.Stdout)
	if !options.Yes && !confirmReset(streams.In, streams.Out, c.Colors.Stdout, logger) {
		return 1
	}
	fmt.Fprintln(streams.Out, ui.Paint(c.Colors.Stdout, "Removing worktrees...", ui.Bold))
	if index, err := runWithConcurrencyLimit(removals, resetRemoveConcurrency, func(item worktree.Worktree, out, errOut io.Writer) error {
		return c.Client.RemoveWorktree(ctx, item.Path, true, out, errOut)
	}, io.Discard, streams.Err); err != nil {
		logger.Error("remove worktree %s: %v", removals[index].Path, err)
		return 1
	}
	fmt.Fprintln(streams.Out, ui.Paint(c.Colors.Stdout, "Deleting local branches...", ui.Bold))
	if err := c.Client.DeleteBranches(ctx, deletes, io.Discard, streams.Err); err != nil {
		logger.Error("delete branches: %v", err)
		return 1
	}
	fmt.Fprintln(streams.Out, ui.Paint(c.Colors.Stdout, "Adding worktrees...", ui.Bold))
	if index, err := runWithConcurrencyLimit(cfg.Worktrees, resetAddConcurrency, func(item worktree.ConfiguredItem, out, errOut io.Writer) error {
		return c.Client.AddWorktree(ctx, item, defaultBranch, out, errOut)
	}, io.Discard, streams.Err); err != nil {
		logger.Error("add worktree %s: %v", cfg.Worktrees[index].Path, err)
		return 1
	}
	fmt.Fprintln(streams.Out, ui.Paint(c.Colors.Stdout, "Branches", ui.Bold))
	if err := c.Client.ListRawBranches(ctx, streams.Out, streams.Err); err != nil {
		logger.Error("list branches: %v", err)
		return 1
	}
	return 0
}

type concurrentOperationResult struct {
	index          int
	stdout, stderr string
	err            error
}

func runWithConcurrencyLimit[T any](items []T, limit int, operation func(T, io.Writer, io.Writer) error, stdout, stderr io.Writer) (int, error) {
	results := make([]concurrentOperationResult, len(items))
	completed := make(chan concurrentOperationResult, limit)
	next, active := 0, 0
	failed := false
	start := func(index int) {
		active++
		go func() {
			out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
			err := operation(items[index], out, errOut)
			completed <- concurrentOperationResult{index, out.String(), errOut.String(), err}
		}()
	}
	for active < limit && next < len(items) {
		start(next)
		next++
	}
	for active > 0 {
		result := <-completed
		results[result.index] = result
		active--
		if result.err != nil {
			failed = true
		}
		if !failed && next < len(items) {
			start(next)
			next++
		}
	}
	for _, result := range results[:next] {
		_, _ = io.WriteString(stdout, result.stdout)
		_, _ = io.WriteString(stderr, result.stderr)
		if result.err != nil {
			return result.index, result.err
		}
	}
	return -1, nil
}
func showResetTargets(stdout io.Writer, removals []worktree.Worktree, deletes []string, enabled bool) {
	fmt.Fprintln(stdout, ui.Paint(enabled, "grape reset will delete:", ui.Red, ui.Bold))
	fmt.Fprintln(stdout, ui.Paint(enabled, "Worktrees", ui.Blue, ui.Bold))
	if len(removals) == 0 {
		fmt.Fprintln(stdout, ui.Paint(enabled, "  (none)", ui.Dim))
	} else {
		fmt.Fprint(stdout, ui.FormatWorktreeList(removals, enabled))
	}
	fmt.Fprintln(stdout, ui.Paint(enabled, "Branches", ui.Red, ui.Bold))
	if len(deletes) == 0 {
		fmt.Fprintln(stdout, ui.Paint(enabled, "  (none)", ui.Dim))
	} else {
		for _, branch := range deletes {
			fmt.Fprintf(stdout, "  %s\n", ui.Paint(enabled, branch, ui.Green))
		}
	}
}
func showResetCounts(stdout io.Writer, removalCount, additionCount int, enabled bool) {
	fmt.Fprintf(stdout, "%s %d\n", ui.Paint(enabled, "Worktrees to remove:", ui.Bold), removalCount)
	fmt.Fprintf(stdout, "%s %d\n", ui.Paint(enabled, "Worktrees to add:", ui.Bold), additionCount)
}
func confirmReset(stdin io.Reader, stdout io.Writer, enabled bool, logger logging.Logger) bool {
	fmt.Fprint(stdout, ui.Paint(enabled, "Proceed with reset? [y/N] ", ui.Red, ui.Bold))
	if stdin == nil {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, ui.Paint(enabled, "grape: reset cancelled", ui.Dim))
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
	fmt.Fprintln(stdout, ui.Paint(enabled, "grape: reset cancelled", ui.Dim))
	return false
}
