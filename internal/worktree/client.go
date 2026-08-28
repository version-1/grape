package worktree

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Worktree struct {
	Path   string
	Head   string
	Branch string
	Main   bool
}

type ConfiguredItem struct {
	Path       string `json:"path"`
	Branch     string `json:"branch"`
	StartPoint string `json:"start_point,omitempty"`
}

type Client interface {
	ListWorktrees(context.Context) ([]Worktree, error)
	RemoveWorktree(context.Context, string, bool, io.Writer, io.Writer) error
	DeleteBranch(context.Context, string, io.Writer, io.Writer) error
	DeleteBranches(context.Context, []string, io.Writer, io.Writer) error
	ListBranches(context.Context) ([]string, error)
	ListRawBranches(context.Context, io.Writer, io.Writer) error
	DefaultBranch(context.Context) (string, error)
	AddWorktree(context.Context, ConfiguredItem, string, io.Writer, io.Writer) error
	CurrentBranch(context.Context) (string, error)
	RebaseBranch(context.Context) (string, error)
	ValidateBranch(context.Context, string) error
	OriginURL(context.Context) (string, error)
	Push(context.Context, string, bool, io.Writer, io.Writer) error
	Rebase(context.Context, []string, io.Reader, io.Writer, io.Writer) error
}

type RunGitFunc func(context.Context, []string, io.Reader, io.Writer, io.Writer) error

type CommandClient struct {
	RunCommand RunGitFunc
	ReadFile   func(string) ([]byte, error)
}

func (c CommandClient) run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if c.RunCommand != nil {
		return c.RunCommand(ctx, args, stdin, stdout, stderr)
	}
	return RunGit(ctx, args, stdin, stdout, stderr)
}

func (c CommandClient) Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmdArgs := append([]string{"worktree"}, args...)
	return c.run(ctx, cmdArgs, stdin, stdout, stderr)
}

func (c CommandClient) ListWorktrees(ctx context.Context) ([]Worktree, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"worktree", "list", "--porcelain"}, nil, stdout, stderr); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return Parse(stdout.String()), nil
}

func (c CommandClient) RemoveWorktree(ctx context.Context, path string, force bool, stdout io.Writer, stderr io.Writer) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	return c.run(ctx, args, nil, stdout, stderr)
}

func (c CommandClient) DeleteBranch(ctx context.Context, branch string, stdout io.Writer, stderr io.Writer) error {
	return c.DeleteBranches(ctx, []string{branch}, stdout, stderr)
}

func (c CommandClient) DeleteBranches(ctx context.Context, branches []string, stdout io.Writer, stderr io.Writer) error {
	if len(branches) == 0 {
		return nil
	}
	args := append([]string{"branch", "-D"}, branches...)
	return c.run(ctx, args, nil, stdout, stderr)
}

func (c CommandClient) ListBranches(ctx context.Context) ([]string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"branch", "--format=%(refname:short)"}, nil, stdout, stderr); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	branches := make([]string, 0)
	for _, line := range strings.Split(stdout.String(), "\n") {
		branch := strings.TrimSpace(line)
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	return branches, nil
}

func (c CommandClient) ListRawBranches(ctx context.Context, stdout io.Writer, stderr io.Writer) error {
	return c.run(ctx, []string{"branch"}, nil, stdout, stderr)
}

func (c CommandClient) DefaultBranch(ctx context.Context) (string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"symbolic-ref", "--short", "refs/remotes/origin/HEAD"}, nil, stdout, stderr); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return strings.TrimPrefix(strings.TrimSpace(stdout.String()), "origin/"), nil
}

func (c CommandClient) AddWorktree(ctx context.Context, item ConfiguredItem, defaultBranch string, stdout io.Writer, stderr io.Writer) error {
	startPoint := item.StartPoint
	if startPoint == "" {
		startPoint = defaultBranch
	}
	return c.run(ctx, []string{"worktree", "add", "-B", item.Branch, item.Path, startPoint}, nil, stdout, stderr)
}

func (c CommandClient) CurrentBranch(ctx context.Context) (string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, nil, stdout, stderr); err != nil {
		return "", commandError(err, stderr)
	}
	branch := strings.TrimSpace(stdout.String())
	if branch == "" {
		return "", errors.New("current branch is empty")
	}
	return branch, nil
}

func (c CommandClient) RebaseBranch(ctx context.Context) (string, error) {
	readFile := c.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}

	for _, stateDirectory := range []string{"rebase-merge", "rebase-apply"} {
		path, err := c.gitPath(ctx, stateDirectory+"/head-name")
		if err != nil {
			return "", err
		}
		contents, err := readFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}

		const branchPrefix = "refs/heads/"
		branch := strings.TrimSpace(string(contents))
		if !strings.HasPrefix(branch, branchPrefix) || branch == branchPrefix {
			return "", fmt.Errorf("rebase head-name is not a local branch: %q", branch)
		}
		return strings.TrimPrefix(branch, branchPrefix), nil
	}
	return "", errors.New("rebase is not in progress")
}

func (c CommandClient) gitPath(ctx context.Context, path string) (string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"rev-parse", "--git-path", path}, nil, stdout, stderr); err != nil {
		return "", commandError(err, stderr)
	}
	resolved := strings.TrimSpace(stdout.String())
	if resolved == "" {
		return "", errors.New("git path is empty")
	}
	return resolved, nil
}

func (c CommandClient) ValidateBranch(ctx context.Context, branch string) error {
	if strings.HasPrefix(branch, "-") {
		return fmt.Errorf("branch name must not begin with '-': %q", branch)
	}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"check-ref-format", "--branch", branch}, nil, io.Discard, stderr); err != nil {
		return commandError(err, stderr)
	}
	return nil
}

func (c CommandClient) OriginURL(ctx context.Context) (string, error) {
	pushURLs, err := c.configValues(ctx, "remote.origin.pushurl")
	if err != nil {
		return "", err
	}
	if len(pushURLs) > 0 {
		return exactlyOneURL("remote.origin.pushurl", pushURLs)
	}

	urls, err := c.configValues(ctx, "remote.origin.url")
	if err != nil {
		return "", err
	}
	return exactlyOneURL("remote.origin.url", urls)
}

func (c CommandClient) configValues(ctx context.Context, key string) ([]string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := c.run(ctx, []string{"config", "--get-all", key}, nil, stdout, stderr)
	if err != nil {
		var exitErr interface{ ExitCode() int }
		if !(errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && stdout.Len() == 0 && stderr.Len() == 0) {
			return nil, commandError(err, stderr)
		}
	}

	if err != nil {
		return nil, nil
	}
	output := strings.TrimSuffix(stdout.String(), "\n")
	return strings.Split(output, "\n"), nil
}

func exactlyOneURL(key string, values []string) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("%s is missing", key)
	}
	if len(values) != 1 {
		return "", fmt.Errorf("%s must have exactly one value", key)
	}
	if strings.TrimSpace(values[0]) == "" {
		return "", fmt.Errorf("%s is empty", key)
	}
	return values[0], nil
}

func (c CommandClient) Push(ctx context.Context, branch string, forceWithLease bool, stdout io.Writer, stderr io.Writer) error {
	args := []string{"push"}
	if forceWithLease {
		args = append(args, "--force-with-lease")
	}
	args = append(args, "origin", "HEAD:"+branch)
	return c.run(ctx, args, nil, stdout, stderr)
}

func (c CommandClient) Rebase(ctx context.Context, rebaseArgs []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	args := append([]string{"rebase"}, rebaseArgs...)
	return c.run(ctx, args, stdin, stdout, stderr)
}

func commandError(err error, stderr *bytes.Buffer) error {
	if stderr.Len() == 0 {
		return err
	}
	return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
}

func RunGit(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
