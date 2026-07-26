package worktree

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	RemoveWorktree(context.Context, string, io.Writer, io.Writer) error
	DeleteBranch(context.Context, string, io.Writer, io.Writer) error
	ListBranches(context.Context) ([]string, error)
	DefaultBranch(context.Context) (string, error)
	AddWorktree(context.Context, ConfiguredItem, string, io.Writer, io.Writer) error
}

type CommandClient struct{}

func (CommandClient) Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmdArgs := append([]string{"worktree"}, args...)
	return RunGit(ctx, cmdArgs, stdin, stdout, stderr)
}

func (CommandClient) ListWorktrees(ctx context.Context) ([]Worktree, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := RunGit(ctx, []string{"worktree", "list", "--porcelain"}, nil, stdout, stderr); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return Parse(stdout.String()), nil
}

func (CommandClient) RemoveWorktree(ctx context.Context, path string, stdout io.Writer, stderr io.Writer) error {
	return RunGit(ctx, []string{"worktree", "remove", path}, nil, stdout, stderr)
}

func (CommandClient) DeleteBranch(ctx context.Context, branch string, stdout io.Writer, stderr io.Writer) error {
	return RunGit(ctx, []string{"branch", "-D", branch}, nil, stdout, stderr)
}

func (CommandClient) ListBranches(ctx context.Context) ([]string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := RunGit(ctx, []string{"branch", "--format=%(refname:short)"}, nil, stdout, stderr); err != nil {
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

func (CommandClient) DefaultBranch(ctx context.Context) (string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := RunGit(ctx, []string{"symbolic-ref", "--short", "refs/remotes/origin/HEAD"}, nil, stdout, stderr); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return strings.TrimPrefix(strings.TrimSpace(stdout.String()), "origin/"), nil
}

func (CommandClient) AddWorktree(ctx context.Context, item ConfiguredItem, defaultBranch string, stdout io.Writer, stderr io.Writer) error {
	startPoint := item.StartPoint
	if startPoint == "" {
		startPoint = defaultBranch
	}
	return RunGit(ctx, []string{"worktree", "add", "-B", item.Branch, item.Path, startPoint}, nil, stdout, stderr)
}

func RunGit(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
