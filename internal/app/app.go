package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-1/grape/internal/color"
	"github.com/version-1/grape/internal/config"
	"github.com/version-1/grape/internal/logging"
	"github.com/version-1/grape/internal/ui"
	"github.com/version-1/grape/internal/worktree"
)

type WorktreeRunner interface {
	Run(context.Context, []string, io.Reader, io.Writer, io.Writer) error
}

type App struct {
	client            worktree.Client
	runner            WorktreeRunner
	readFile          config.ReadFileFunc
	pathResolver      config.PathResolver
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
	application := App{
		client:            client,
		runner:            runner,
		readFile:          readFile,
		pathResolver:      config.DefaultPathResolver(),
		exampleConfigPath: "grape.example.json",
		version:           "dev",
		commit:            "unknown",
	}
	if readFile == nil {
		application.readFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
		application.pathResolver = config.PathResolver{
			Stat:     func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
			Env:      func(string) string { return "" },
			UserHome: func() (string, error) { return "/nonexistent", nil },
		}
	}
	return application
}

func (a App) WithExampleConfigPath(path string) App {
	a.exampleConfigPath = path
	return a
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
	logger := logging.New(stderr, a.colors.Stderr)

	switch args[0] {
	case "help", "--help", "-h":
		return a.runHelp(stdout)
	case "version":
		return a.runVersion(stdout)
	case "init":
		return a.runInit(args[1:], stdout, logger)
	case "list":
		if len(args) == 1 {
			if _, code := a.readConfig("", logger); code != 0 {
				return code
			}
			return a.runList(ctx, stdout, logger)
		}
	case "branch":
		if _, code := a.readConfig("", logger); code != 0 {
			return code
		}
		return a.runBranch(ctx, args[1:], stdout, logger)
	case "remove":
		if _, code := a.readConfig("", logger); code != 0 {
			return code
		}
		return a.runRemove(ctx, args[1:], stdin, stdout, stderr, logger)
	case "reset":
		return a.runReset(ctx, args[1:], stdin, stdout, stderr, logger)
	case "push":
		cfg, code := a.readConfig("", logger)
		if code != 0 {
			return code
		}
		return a.runPush(ctx, args[1:], cfg, stdout, stderr, logger)
	case "rebase":
		return a.runRebase(ctx, args[1:], stdin, stdout, stderr, logger)
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

func (a App) readConfig(explicitPath string, logger logging.Logger) (config.Config, int) {
	discovery, err := a.pathResolver.Discover(explicitPath)
	if err != nil {
		logger.Error("resolve config: %v", err)
		return config.Config{}, 2
	}
	if !discovery.Found {
		logger.Warning("grape.json not found; using default protected branches: main, master")
		return config.Config{}, 0
	}
	cfg, err := config.Read(discovery.Path, a.readFile)
	if err != nil {
		logger.Error("read config: %v", err)
		return config.Config{}, 2
	}
	return cfg, 0
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
  grape push [--force-with-lease]
  grape rebase [--config|-c <path>] [--] [<git-rebase-args>...]
  grape init
  grape version
  grape help

Commands:
  list      Show worktree path, branch, and HEAD in a colored table.
  branch    Show worktrees that reference the given local branch.
  remove    Remove matching worktrees and their local branches.
  reset     Recreate worktrees from config after removing non-default worktrees and local branches.
  push      Safely push the current branch to origin.
  rebase    Rebase the current branch when allowed by config.
  init      Create grape.json in the resolved config home from grape.example.json.
  version   Show the build version and commit hash.
  help      Show this help.

Config:
  Configuration-aware commands read config in this order:
    1. ./grape.json
    2. $GRAPE_HOME/grape.json
    3. ~/.grape/grape.json
  Config is decoded strictly. Missing config uses protected defaults main and master,
  except rebase, which requires an existing config file.

Safety:
  push accepts no arguments or exactly --force-with-lease, always targets origin,
  and refuses protected branches.
  rebase validates its complete allow policy before running git rebase.
  Except for grape's config option, rebase arguments pass through to git rebase.
  reset and remove never remove the main working tree.
  reset also keeps the branch checked out by the main working tree.
  reset shows deletion targets and continues only after y/yes confirmation.

Unknown commands are delegated to git worktree.

Color is automatic per output stream and disabled by NO_COLOR.
`, "\n")
}

func (a App) runInit(args []string, stdout io.Writer, logger logging.Logger) int {
	if len(args) != 0 {
		logger.Error("usage: grape init")
		return 2
	}

	grapeHome, err := a.pathResolver.GrapeHome()
	if err != nil {
		logger.Error("resolve config directory: %v", err)
		return 1
	}
	destination := filepath.Join(grapeHome, "grape.json")
	if err := config.Initialize(a.exampleConfigPath, destination); err != nil {
		if errors.Is(err, config.ErrConfigAlreadyExists) {
			logger.Error("config already exists: %s", destination)
			return 1
		}
		logger.Error("initialize config: %v", err)
		return 1
	}

	fmt.Fprintf(stdout, "Created config at %s\n", destination)
	return 0
}

func (a App) runList(ctx context.Context, stdout io.Writer, logger logging.Logger) int {
	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("%v", err)
		return 1
	}
	if len(worktrees) == 0 {
		fmt.Fprintln(stdout, ui.Paint(a.colors.Stdout, "No worktrees", ui.Dim))
		return 0
	}

	fmt.Fprint(stdout, ui.FormatWorktreeList(worktrees, a.colors.Stdout))
	return 0
}

func (a App) runBranch(ctx context.Context, args []string, stdout io.Writer, logger logging.Logger) int {
	if len(args) != 1 {
		logger.Error("usage: grape branch <branch-name>")
		return 2
	}

	branch := worktree.NormalizeBranchName(args[0])
	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("%v", err)
		return 1
	}

	targets := worktree.Filter(worktrees, func(item worktree.Worktree) bool {
		return item.Branch == branch
	})
	if len(targets) == 0 {
		fmt.Fprintf(stdout, "%s %s\n", ui.Paint(a.colors.Stdout, "No worktree for branch", ui.Dim), ui.Paint(a.colors.Stdout, branch, ui.Green))
		return 1
	}

	fmt.Fprint(stdout, ui.FormatWorktreeList(targets, a.colors.Stdout))
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

func (a App) runRemove(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, logger logging.Logger) int {
	options, err := parseRemoveOptions(args)
	if err != nil {
		logger.Error("%v", err)
		return 2
	}

	matcher, err := worktree.NewMatcher(worktree.MatchOptions{
		Pattern: options.Pattern,
		Regex:   options.Regex,
	})
	if err != nil {
		logger.Error("%v", err)
		return 2
	}

	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("list worktrees: %v", err)
		return 1
	}

	targets := worktree.Filter(worktrees, func(item worktree.Worktree) bool {
		return !item.Main && matcher(item)
	})
	if len(targets) == 0 {
		fmt.Fprintln(stdout, ui.Paint(a.colors.Stdout, "grape: no matching worktrees", ui.Dim))
		return 0
	}
	if !confirmRemove(stdin, stdout, targets, a.colors.Stdout, logger) {
		return 1
	}

	for _, target := range targets {
		if code := a.removeWorktreeAndBranch(ctx, target, stdout, stderr); code != 0 {
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

func parsePushOptions(args []string) (bool, error) {
	switch {
	case len(args) == 0:
		return false, nil
	case len(args) == 1 && args[0] == "--force-with-lease":
		return true, nil
	default:
		return false, errors.New("usage: grape push [--force-with-lease]")
	}
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
			if index >= len(args) {
				return rebaseOptions{}, fmt.Errorf("%s requires a path", arg)
			}
			options.ConfigPath = args[index]
			continue
		}
		if strings.HasPrefix(arg, "--config=") {
			options.ConfigPath = strings.TrimPrefix(arg, "--config=")
			continue
		}
		options.GitArgs = append(options.GitArgs, arg)
	}
	return options, nil
}

func (a App) runRebase(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, logger logging.Logger) int {
	options, err := parseRebaseOptions(args)
	if err != nil {
		logger.Error("usage: grape rebase [--config|-c <path>] [--] [<git-rebase-args>...]")
		return 2
	}

	discovery, err := a.pathResolver.Discover(options.ConfigPath)
	if err != nil {
		logger.Error("resolve config: %v", err)
		return 2
	}
	if !discovery.Found {
		logger.Error("rebase config not found: %s", discovery.Path)
		return 2
	}
	cfg, err := config.ReadRebaseConfig(discovery.Path, a.readFile)
	if err != nil {
		logger.Error("read config: %v", err)
		return 2
	}

	branch, err := a.client.CurrentBranch(ctx)
	if err != nil {
		branch, err = a.client.RebaseBranch(ctx)
		if err != nil {
			logger.Error("current HEAD is detached outside an active rebase or not in a git repository")
			return 1
		}
	}
	if !cfg.BranchAllowedForRebase(branch) {
		logger.Error("rebase is not allowed for current branch: %s", branch)
		return 1
	}

	if err := a.client.Rebase(ctx, options.GitArgs, stdin, stdout, stderr); err != nil {
		if exitErr, ok := err.(exitCodeError); ok {
			return exitErr.ExitCode()
		}
		logger.Error("%v", err)
		return 1
	}
	return 0
}

func (a App) runPush(ctx context.Context, args []string, cfg config.Config, stdout io.Writer, stderr io.Writer, logger logging.Logger) int {
	forceWithLease, err := parsePushOptions(args)
	if err != nil {
		logger.Error("%v", err)
		return 2
	}

	branch, err := a.client.CurrentBranch(ctx)
	if err != nil {
		logger.Error("current HEAD is detached or not in a git repository")
		return 1
	}
	if cfg.BranchProtected(branch) {
		logger.Error("refusing to push protected branch: %s", branch)
		return 1
	}
	if err := a.client.ValidateBranch(ctx, branch); err != nil {
		if strings.HasPrefix(branch, "-") {
			logger.Error("refusing to push branch that starts with '-': %s", branch)
		} else {
			logger.Error("invalid branch name: %s", branch)
		}
		return 1
	}
	originURL, err := a.client.OriginURL(ctx)
	if err != nil {
		logger.Error("%v", err)
		return 1
	}

	command := "git push origin HEAD:" + branch
	if forceWithLease {
		command = "git push --force-with-lease origin HEAD:" + branch
	}
	logger.Info("branch: %s", branch)
	logger.Info("origin: %s", originURL)
	logger.Info("running: %s", command)

	if err := a.client.Push(ctx, branch, forceWithLease, stdout, stderr); err != nil {
		if exitErr, ok := err.(exitCodeError); ok {
			return exitErr.ExitCode()
		}
		logger.Error("%v", err)
		return 1
	}
	return 0
}

func (a App) runReset(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, logger logging.Logger) int {
	options, err := parseResetOptions(args)
	if err != nil {
		logger.Error("%v", err)
		return 2
	}

	resetConfig, code := a.readConfig(options.ConfigPath, logger)
	if code != 0 {
		return code
	}
	if err := resetConfig.ValidateReset(); err != nil {
		logger.Error("read config: %v", err)
		return 2
	}

	defaultBranch := resetConfig.DefaultBranch
	if defaultBranch == "" {
		defaultBranch, err = a.client.DefaultBranch(ctx)
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

	worktrees, err := a.client.ListWorktrees(ctx)
	if err != nil {
		logger.Error("list worktrees: %v", err)
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
		logger.Error("list branches: %v", err)
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

	if !confirmReset(stdin, stdout, removeTargets, deleteTargets, a.colors.Stdout, logger) {
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
		fmt.Fprintf(stdout, "%s %s %s\n", ui.Paint(a.colors.Stdout, "Adding worktree", ui.Green, ui.Bold), ui.Paint(a.colors.Stdout, item.Path, ui.Cyan), ui.Paint(a.colors.Stdout, item.Branch, ui.Green))
		if err := a.client.AddWorktree(ctx, item, defaultBranch, stdout, stderr); err != nil {
			logger.Error("add worktree %s: %v", item.Path, err)
			return 1
		}
	}

	return 0
}

func confirmReset(stdin io.Reader, stdout io.Writer, removeTargets []worktree.Worktree, deleteTargets []string, colorEnabled bool, logger logging.Logger) bool {
	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "grape reset will delete:", ui.Red, ui.Bold))
	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "Worktrees", ui.Blue, ui.Bold))
	if len(removeTargets) == 0 {
		fmt.Fprintln(stdout, ui.Paint(colorEnabled, "  (none)", ui.Dim))
	} else {
		fmt.Fprint(stdout, ui.FormatWorktreeList(removeTargets, colorEnabled))
	}

	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "Branches", ui.Red, ui.Bold))
	if len(deleteTargets) == 0 {
		fmt.Fprintln(stdout, ui.Paint(colorEnabled, "  (none)", ui.Dim))
	} else {
		for _, branch := range deleteTargets {
			fmt.Fprintf(stdout, "  %s\n", ui.Paint(colorEnabled, branch, ui.Green))
		}
	}

	fmt.Fprint(stdout, ui.Paint(colorEnabled, "Proceed with reset? [y/N] ", ui.Red, ui.Bold))
	if stdin == nil {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, ui.Paint(colorEnabled, "grape: reset cancelled", ui.Dim))
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

	fmt.Fprintln(stdout, ui.Paint(colorEnabled, "grape: reset cancelled", ui.Dim))
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
	fmt.Fprintf(stdout, "%s %s\n", ui.Paint(a.colors.Stdout, "Removing worktree", ui.Blue, ui.Bold), ui.Paint(a.colors.Stdout, item.Path, ui.Cyan))
	if err := a.client.RemoveWorktree(ctx, item.Path, stdout, stderr); err != nil {
		logging.New(stderr, a.colors.Stderr).Error("remove worktree %s: %v", item.Path, err)
		return 1
	}
	return 0
}

func (a App) deleteBranch(ctx context.Context, branch string, stdout io.Writer, stderr io.Writer) int {
	fmt.Fprintf(stdout, "%s %s\n", ui.Paint(a.colors.Stdout, "Deleting branch", ui.Red, ui.Bold), ui.Paint(a.colors.Stdout, branch, ui.Green))
	if err := a.client.DeleteBranch(ctx, branch, stdout, stderr); err != nil {
		logging.New(stderr, a.colors.Stderr).Error("delete branch %s: %v", branch, err)
		return 1
	}
	return 0
}
