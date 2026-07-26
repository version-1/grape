package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/version-1/dotfiles/shared/commands/gw/internal/config"
	"github.com/version-1/dotfiles/shared/commands/gw/internal/ui"
	"github.com/version-1/dotfiles/shared/commands/gw/internal/worktree"
)

type WorktreeRunner interface {
	Run(context.Context, []string, io.Reader, io.Writer, io.Writer) error
}

type App struct {
	client       worktree.Client
	runner       WorktreeRunner
	readFile     config.ReadFileFunc
	pathResolver config.PathResolver
	version      string
	commit       string
}

func New(client worktree.Client, runner WorktreeRunner, readFile config.ReadFileFunc) App {
	return App{
		client:       client,
		runner:       runner,
		readFile:     readFile,
		pathResolver: config.DefaultPathResolver(),
		version:      "dev",
		commit:       "unknown",
	}
}

func (a App) WithPathResolver(pathResolver config.PathResolver) App {
	a.pathResolver = pathResolver
	return a
}

func (a App) WithBuildInfo(version string, commit string) App {
	a.version = version
	a.commit = commit
	return a
}

type exitCodeError interface {
	ExitCode() int
}

func (a App) Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"list"}
	}

	switch args[0] {
	case "help", "--help", "-h":
		return a.runHelp(stdout)
	case "version":
		return a.runVersion(stdout)
	case "list":
		if len(args) == 1 {
			return a.runList(ctx, stdout, stderr)
		}
	case "branch":
		return a.runBranch(ctx, args[1:], stdout, stderr)
	case "remove":
		return a.runRemove(ctx, args[1:], stdin, stdout, stderr)
	case "reset":
		return a.runReset(ctx, args[1:], stdin, stdout, stderr)
	}

	if err := a.runner.Run(ctx, args, stdin, stdout, stderr); err != nil {
		if exitErr, ok := err.(exitCodeError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "grape: %v\n", err)
		return 1
	}
	return 0
}

func (a App) runHelp(stdout io.Writer) int {
	fmt.Fprint(stdout, helpText())
	return 0
}

func (a App) runVersion(stdout io.Writer) int {
	fmt.Fprintf(stdout, "grape %s (%s)\n", a.version, a.commit)
	return 0
}

func helpText() string {
	return strings.TrimLeft(`
grape is a Go CLI for git worktree workflows.

Usage:
  grape
  grape list
  grape branch <branch-name>
  grape remove [--regex|-r] <path-prefix-or-pattern>
  grape reset [--config|-c <path>]
	grape version
  grape help

Commands:
  list      Show worktree path, branch, and HEAD in a colored table.
  branch    Show worktrees that reference the given local branch.
  remove    Remove matching worktrees and their local branches.
  reset     Recreate worktrees from config after removing non-default worktrees and local branches.
	version   Show the build version and commit hash.
  help      Show this help.

Config:
  grape reset reads config in this order when --config is omitted:
    1. ./gw.json
    2. $GW_HOME/gw.json
    3. ~/.gw/gw.json

Safety:
  reset and remove never remove the main working tree.
  reset also keeps the branch checked out by the main working tree.
  reset shows deletion targets and continues only after y/yes confirmation.

Unknown commands are delegated to git worktree.
`, "\n")
}

func (a App) runList(ctx context.Context, stdout io.Writer, stderr io.Writer) int {
	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "%s %s\n", ui.Paint("grape:", ui.Red, ui.Bold), err)
		return 1
	}
	if len(worktrees) == 0 {
		fmt.Fprintln(stdout, ui.Paint("No worktrees", ui.Dim))
		return 0
	}

	fmt.Fprint(stdout, ui.FormatWorktreeList(worktrees))
	return 0
}

func (a App) runBranch(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "%s usage: grape branch <branch-name>\n", ui.Paint("grape:", ui.Red, ui.Bold))
		return 2
	}

	branch := worktree.NormalizeBranchName(args[0])
	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "%s %s\n", ui.Paint("grape:", ui.Red, ui.Bold), err)
		return 1
	}

	targets := worktree.Filter(worktrees, func(item worktree.Worktree) bool {
		return item.Branch == branch
	})
	if len(targets) == 0 {
		fmt.Fprintf(stdout, "%s %s\n", ui.Paint("No worktree for branch", ui.Dim), ui.Paint(branch, ui.Green))
		return 1
	}

	fmt.Fprint(stdout, ui.FormatWorktreeList(targets))
	return 0
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
	return removeOptions{
		Pattern: flags.Arg(0),
		Regex:   *regex,
	}, nil
}

func (a App) runRemove(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	options, err := parseRemoveOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "grape: %v\n", err)
		return 2
	}

	matcher, err := worktree.NewMatcher(worktree.MatchOptions{
		Pattern: options.Pattern,
		Regex:   options.Regex,
	})
	if err != nil {
		fmt.Fprintf(stderr, "grape: %v\n", err)
		return 2
	}

	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "grape: list worktrees: %v\n", err)
		return 1
	}

	targets := worktree.Filter(worktrees, func(item worktree.Worktree) bool {
		return !item.Main && matcher(item)
	})
	if len(targets) == 0 {
		fmt.Fprintln(stdout, ui.Paint("grape: no matching worktrees", ui.Dim))
		return 0
	}
	if !confirmRemove(stdin, stdout, targets) {
		return 1
	}

	for _, target := range targets {
		if code := a.removeWorktreeAndBranch(ctx, target, stdout, stderr); code != 0 {
			return code
		}
	}
	return 0
}

func confirmRemove(stdin io.Reader, stdout io.Writer, targets []worktree.Worktree) bool {
	fmt.Fprintln(stdout, ui.Paint("grape remove will delete:", ui.Red, ui.Bold))
	fmt.Fprintln(stdout, ui.Paint("Worktrees", ui.Blue, ui.Bold))
	fmt.Fprint(stdout, ui.FormatWorktreeList(targets))
	fmt.Fprintln(stdout, ui.Paint("Branches", ui.Red, ui.Bold))
	for _, target := range targets {
		if target.Branch != "" {
			fmt.Fprintf(stdout, "  %s\n", ui.Paint(target.Branch, ui.Green))
		}
	}

	fmt.Fprint(stdout, ui.Paint("Proceed with removal? [y/N] ", ui.Red, ui.Bold))
	if stdin == nil {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, ui.Paint("grape: removal cancelled", ui.Dim))
		return false
	}

	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stdout, "%s %v\n", ui.Paint("grape: read confirmation:", ui.Red, ui.Bold), err)
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "y" || answer == "yes" {
		return true
	}

	fmt.Fprintln(stdout, ui.Paint("grape: removal cancelled", ui.Dim))
	return false
}

type resetOptions struct {
	ConfigPath string
}

func parseResetOptions(args []string) (resetOptions, error) {
	flags := flag.NewFlagSet("reset", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "path to grape reset config")
	flags.StringVar(configPath, "c", "", "path to grape reset config")

	if err := flags.Parse(args); err != nil {
		return resetOptions{}, err
	}
	if flags.NArg() != 0 {
		return resetOptions{}, errors.New("usage: grape reset [--config|-c <path>]")
	}
	return resetOptions{ConfigPath: *configPath}, nil
}

func (a App) runReset(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	options, err := parseResetOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "grape: %v\n", err)
		return 2
	}

	configPath, err := a.pathResolver.ResolveConfigPath(options.ConfigPath)
	if err != nil {
		fmt.Fprintf(stderr, "grape: resolve config: %v\n", err)
		return 2
	}

	resetConfig, err := config.ReadResetConfig(configPath, a.readFile)
	if err != nil {
		fmt.Fprintf(stderr, "grape: read config: %v\n", err)
		return 2
	}

	defaultBranch := resetConfig.DefaultBranch
	if defaultBranch == "" {
		defaultBranch, err = a.client.DefaultBranch(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "grape: detect default branch: %v\n", err)
			return 1
		}
	}
	defaultBranch = worktree.NormalizeBranchName(defaultBranch)
	if defaultBranch == "" {
		fmt.Fprintln(stderr, "grape: default branch is empty")
		return 2
	}

	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "grape: list worktrees: %v\n", err)
		return 1
	}

	protectedBranches := map[string]struct{}{
		defaultBranch: {},
	}
	removeTargets := make([]worktree.Worktree, 0)
	for _, item := range worktrees {
		if item.Main || item.Branch == defaultBranch {
			if item.Branch != "" {
				protectedBranches[item.Branch] = struct{}{}
			}
			continue
		}
		removeTargets = append(removeTargets, item)
	}

	branches, err := a.client.ListBranches(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "grape: list branches: %v\n", err)
		return 1
	}

	deleteTargets := make([]string, 0)
	for _, branch := range branches {
		branch = worktree.NormalizeBranchName(branch)
		if branch == "" {
			continue
		}
		if _, ok := protectedBranches[branch]; ok {
			continue
		}
		deleteTargets = append(deleteTargets, branch)
	}

	if !confirmReset(stdin, stdout, removeTargets, deleteTargets) {
		return 1
	}

	for _, item := range removeTargets {
		if code := a.removeWorktreeOnly(ctx, item, stdout, stderr); code != 0 {
			return code
		}
	}

	for _, branch := range deleteTargets {
		if code := a.deleteBranch(ctx, branch, stdout, stderr); code != 0 {
			return code
		}
	}

	for _, item := range resetConfig.Worktrees {
		fmt.Fprintf(stdout, "%s %s %s\n", ui.Paint("Adding worktree", ui.Green, ui.Bold), ui.Paint(item.Path, ui.Cyan), ui.Paint(item.Branch, ui.Green))
		if err := a.client.AddWorktree(ctx, item, defaultBranch, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "%s add worktree %s: %v\n", ui.Paint("grape:", ui.Red, ui.Bold), item.Path, err)
			return 1
		}
	}

	return 0
}

func confirmReset(stdin io.Reader, stdout io.Writer, removeTargets []worktree.Worktree, deleteTargets []string) bool {
	fmt.Fprintln(stdout, ui.Paint("grape reset will delete:", ui.Red, ui.Bold))
	fmt.Fprintln(stdout, ui.Paint("Worktrees", ui.Blue, ui.Bold))
	if len(removeTargets) == 0 {
		fmt.Fprintln(stdout, ui.Paint("  (none)", ui.Dim))
	} else {
		fmt.Fprint(stdout, ui.FormatWorktreeList(removeTargets))
	}

	fmt.Fprintln(stdout, ui.Paint("Branches", ui.Red, ui.Bold))
	if len(deleteTargets) == 0 {
		fmt.Fprintln(stdout, ui.Paint("  (none)", ui.Dim))
	} else {
		for _, branch := range deleteTargets {
			fmt.Fprintf(stdout, "  %s\n", ui.Paint(branch, ui.Green))
		}
	}

	fmt.Fprint(stdout, ui.Paint("Proceed with reset? [y/N] ", ui.Red, ui.Bold))
	if stdin == nil {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, ui.Paint("grape: reset cancelled", ui.Dim))
		return false
	}

	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stdout, "%s %v\n", ui.Paint("grape: read confirmation:", ui.Red, ui.Bold), err)
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "y" || answer == "yes" {
		return true
	}

	fmt.Fprintln(stdout, ui.Paint("grape: reset cancelled", ui.Dim))
	return false
}

func (a App) removeWorktreeAndBranch(ctx context.Context, item worktree.Worktree, stdout io.Writer, stderr io.Writer) int {
	if code := a.removeWorktreeOnly(ctx, item, stdout, stderr); code != 0 {
		return code
	}
	if item.Branch == "" {
		return 0
	}
	return a.deleteBranch(ctx, item.Branch, stdout, stderr)
}

func (a App) removeWorktreeOnly(ctx context.Context, item worktree.Worktree, stdout io.Writer, stderr io.Writer) int {
	fmt.Fprintf(stdout, "%s %s\n", ui.Paint("Removing worktree", ui.Blue, ui.Bold), ui.Paint(item.Path, ui.Cyan))
	if err := a.client.RemoveWorktree(ctx, item.Path, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "%s remove worktree %s: %v\n", ui.Paint("grape:", ui.Red, ui.Bold), item.Path, err)
		return 1
	}
	return 0
}

func (a App) deleteBranch(ctx context.Context, branch string, stdout io.Writer, stderr io.Writer) int {
	fmt.Fprintf(stdout, "%s %s\n", ui.Paint("Deleting branch", ui.Red, ui.Bold), ui.Paint(branch, ui.Green))
	if err := a.client.DeleteBranch(ctx, branch, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "%s delete branch %s: %v\n", ui.Paint("grape:", ui.Red, ui.Bold), branch, err)
		return 1
	}
	return 0
}
