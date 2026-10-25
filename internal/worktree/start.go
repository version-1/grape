package worktree

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// StartBranch refreshes origin, fast-forwards the selected default branch, and
// creates a new branch from it in the current worktree.
func (c CommandClient) StartBranch(ctx context.Context, defaultBranch, branch string, stdout io.Writer, stderr io.Writer) error {
	if defaultBranch == branch {
		return fmt.Errorf("new branch must not be the default branch: %s", branch)
	}
	if err := c.requireCleanWorktree(ctx); err != nil {
		return err
	}
	if err := c.branchExists(ctx, branch); err != nil {
		return err
	}
	remoteDefault := "refs/remotes/origin/" + defaultBranch
	refspec := "+refs/heads/" + defaultBranch + ":" + remoteDefault
	if err := c.run(ctx, []string{"fetch", "origin", refspec}, nil, stdout, stderr); err != nil {
		return err
	}
	if err := c.requireRevision(ctx, remoteDefault); err != nil {
		return fmt.Errorf("remote default branch %q is unavailable: %w", remoteDefault, err)
	}
	if err := c.switchDefaultBranch(ctx, defaultBranch, remoteDefault, stdout, stderr); err != nil {
		return err
	}
	return c.run(ctx, []string{"switch", "-c", branch, "refs/heads/" + defaultBranch}, nil, stdout, stderr)
}

func (c CommandClient) requireCleanWorktree(ctx context.Context) error {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := c.run(ctx, []string{"status", "--porcelain", "--untracked-files=all"}, nil, stdout, stderr); err != nil {
		return commandError(err, stderr)
	}
	if stdout.Len() > 0 {
		return errors.New("working tree has uncommitted or untracked changes")
	}
	for _, state := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"} {
		if err := c.run(ctx, []string{"rev-parse", "--verify", "--quiet", state}, nil, io.Discard, io.Discard); err == nil {
			return fmt.Errorf("cannot start a branch while %s is in progress", strings.ToLower(strings.TrimSuffix(state, "_HEAD")))
		}
	}
	if _, err := c.RebaseBranch(ctx); err == nil {
		return errors.New("cannot start a branch while rebase is in progress")
	} else if err.Error() != "rebase is not in progress" {
		return err
	}
	return nil
}

func (c CommandClient) branchExists(ctx context.Context, branch string) error {
	err := c.run(ctx, []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branch}, nil, io.Discard, io.Discard)
	if err == nil {
		return fmt.Errorf("branch already exists: %s", branch)
	}
	if code, ok := gitExitCode(err); !ok || code != 1 {
		return err
	}
	return nil
}

func (c CommandClient) requireRevision(ctx context.Context, revision string) error {
	return c.run(ctx, []string{"rev-parse", "--verify", "--quiet", revision}, nil, io.Discard, io.Discard)
}

func (c CommandClient) switchDefaultBranch(ctx context.Context, defaultBranch, remoteDefault string, stdout io.Writer, stderr io.Writer) error {
	if err := c.requireRevision(ctx, "refs/heads/"+defaultBranch); err != nil {
		if code, ok := gitExitCode(err); !ok || code != 1 {
			return err
		}
		return c.run(ctx, []string{"switch", "--no-track", "-c", defaultBranch, remoteDefault}, nil, stdout, stderr)
	}
	if err := c.run(ctx, []string{"merge-base", "--is-ancestor", "refs/heads/" + defaultBranch, remoteDefault}, nil, io.Discard, io.Discard); err != nil {
		if code, ok := gitExitCode(err); ok && code == 1 {
			return fmt.Errorf("local default branch %q contains commits not in origin", defaultBranch)
		}
		return err
	}
	if err := c.run(ctx, []string{"switch", defaultBranch}, nil, stdout, stderr); err != nil {
		return err
	}
	return c.run(ctx, []string{"merge", "--ff-only", remoteDefault}, nil, stdout, stderr)
}

func gitExitCode(err error) (int, bool) {
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), true
	}
	return 0, false
}
