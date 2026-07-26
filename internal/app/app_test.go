package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/version-1/grape/internal/config"
	"github.com/version-1/grape/internal/worktree"
)

type fakeRunner struct {
	args   []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	err    error
}

func (r *fakeRunner) Run(_ context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	r.args = append([]string(nil), args...)
	r.stdin = stdin
	r.stdout = stdout
	r.stderr = stderr
	return r.err
}

type fakeExitError struct {
	code int
}

func (e fakeExitError) Error() string {
	return "exit"
}

func (e fakeExitError) ExitCode() int {
	return e.code
}

type fakeClient struct {
	worktrees     []worktree.Worktree
	branches      []string
	defaultBranch string
	listErr       error
	removeErr     error
	deleteErr     error
	addErr        error
	removed       []string
	deleted       []string
	added         []worktree.ConfiguredItem
	addedDefaults []string
}

func (c *fakeClient) ListWorktrees(context.Context) ([]worktree.Worktree, error) {
	return c.worktrees, c.listErr
}

func (c *fakeClient) RemoveWorktree(_ context.Context, path string, _ io.Writer, _ io.Writer) error {
	c.removed = append(c.removed, path)
	return c.removeErr
}

func (c *fakeClient) DeleteBranch(_ context.Context, branch string, _ io.Writer, _ io.Writer) error {
	c.deleted = append(c.deleted, branch)
	return c.deleteErr
}

func (c *fakeClient) ListBranches(context.Context) ([]string, error) {
	return c.branches, c.listErr
}

func (c *fakeClient) DefaultBranch(context.Context) (string, error) {
	return c.defaultBranch, c.listErr
}

func (c *fakeClient) AddWorktree(_ context.Context, item worktree.ConfiguredItem, defaultBranch string, _ io.Writer, _ io.Writer) error {
	c.added = append(c.added, item)
	c.addedDefaults = append(c.addedDefaults, defaultBranch)
	return c.addErr
}

func TestRunDelegatesUnknownWorktreeArgs(t *testing.T) {
	stdin := bytes.NewBufferString("input")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := &fakeRunner{}
	app := New(&fakeClient{}, runner, nil)

	code := app.Run(context.Background(), []string{"list", "--porcelain"}, stdin, stdout, stderr)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !slices.Equal(runner.args, []string{"list", "--porcelain"}) {
		t.Fatalf("args = %#v, want %#v", runner.args, []string{"list", "--porcelain"})
	}
	if runner.stdin != stdin {
		t.Fatal("stdin was not passed to runner")
	}
	if runner.stdout != stdout {
		t.Fatal("stdout was not passed to runner")
	}
	if runner.stderr != stderr {
		t.Fatal("stderr was not passed to runner")
	}
}

func TestRunReturnsRunnerExitCode(t *testing.T) {
	app := New(&fakeClient{}, &fakeRunner{err: fakeExitError{code: 42}}, nil)

	code := app.Run(context.Background(), []string{"prune"}, nil, io.Discard, io.Discard)

	if code != 42 {
		t.Fatalf("code = %d, want 42", code)
	}
}

func TestRunWritesUnexpectedErrors(t *testing.T) {
	stderr := &bytes.Buffer{}
	app := New(&fakeClient{}, &fakeRunner{err: errors.New("failed")}, nil)

	code := app.Run(context.Background(), []string{"prune"}, nil, io.Discard, stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if got, want := stderr.String(), "grape: failed\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestRunHelpShowsInternalHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	runner := &fakeRunner{}
	app := New(&fakeClient{}, runner, nil)

	code := app.Run(context.Background(), []string{"help"}, nil, stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	assertContains(t, stdout.String(), "Usage:")
	assertContains(t, stdout.String(), "grape reset [--config|-c <path>]")
	assertContains(t, stdout.String(), "$GRAPE_HOME/grape.json")
	if runner.args != nil {
		t.Fatalf("runner args = %#v, want nil", runner.args)
	}
}

func TestRunHelpFlagShowsInternalHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	runner := &fakeRunner{}
	app := New(&fakeClient{}, runner, nil)

	code := app.Run(context.Background(), []string{"--help"}, nil, stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	assertContains(t, stdout.String(), "grape is a Go CLI")
	if runner.args != nil {
		t.Fatalf("runner args = %#v, want nil", runner.args)
	}
}

func TestRunVersionShowsBuildInfo(t *testing.T) {
	stdout := &bytes.Buffer{}
	app := New(&fakeClient{}, &fakeRunner{}, nil).WithBuildInfo("0.1.0", "abc1234")

	code := app.Run(context.Background(), []string{"version"}, nil, stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got, want := stdout.String(), "grape 0.1.0 (abc1234)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunListFormatsWorktrees(t *testing.T) {
	stdout := &bytes.Buffer{}
	app := New(&fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Head: "1111111111111111111111111111111111111111", Branch: "main"},
			{Path: "/repo-detached", Head: "2222222222222222222222222222222222222222"},
		},
	}, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"list"}, nil, stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	assertContains(t, stdout.String(), "PATH")
	assertContains(t, stdout.String(), "BRANCH")
	assertContains(t, stdout.String(), "/repo-detached")
	assertContains(t, stdout.String(), "(detached)")
	assertContains(t, stdout.String(), "111111111111")
}

func TestRunBranchShowsMatchingWorktree(t *testing.T) {
	stdout := &bytes.Buffer{}
	app := New(&fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Head: "1111111111111111111111111111111111111111", Branch: "main"},
			{Path: "/repo-feature", Head: "2222222222222222222222222222222222222222", Branch: "feature/example"},
		},
	}, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"branch", "refs/heads/feature/example"}, nil, stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	assertContains(t, stdout.String(), "/repo-feature")
	assertContains(t, stdout.String(), "feature/example")
	if strings.Contains(stdout.String(), "/repo ") {
		t.Fatalf("stdout includes non-matching worktree: %q", stdout.String())
	}
}

func TestRunBranchReturnsOneWhenBranchHasNoWorktree(t *testing.T) {
	stdout := &bytes.Buffer{}
	app := New(&fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main"},
		},
	}, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"branch", "feature/missing"}, nil, stdout, io.Discard)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	assertContains(t, stdout.String(), "No worktree for branch")
	assertContains(t, stdout.String(), "feature/missing")
}

func TestRunRemoveRegexDeletesMatchingWorktrees(t *testing.T) {
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo-feature-one", Branch: "feature/one"},
			{Path: "/repo-feature-two"},
			{Path: "/repo-main", Branch: "main"},
		},
	}
	app := New(client, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"remove", "--regex", `feature-(one|two)$`}, strings.NewReader("yes\n"), io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !slices.Equal(client.removed, []string{"/repo-feature-one", "/repo-feature-two"}) {
		t.Fatalf("removed = %#v", client.removed)
	}
	if !slices.Equal(client.deleted, []string{"feature/one"}) {
		t.Fatalf("deleted = %#v", client.deleted)
	}
}

func TestRunRemoveSkipsMainWorktree(t *testing.T) {
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "feature/current", Main: true},
			{Path: "/repo-feature", Branch: "feature/old"},
		},
	}
	app := New(client, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"remove", "--regex", `/repo.*$`}, strings.NewReader("y\n"), io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !slices.Equal(client.removed, []string{"/repo-feature"}) {
		t.Fatalf("removed = %#v, want %#v", client.removed, []string{"/repo-feature"})
	}
	if !slices.Equal(client.deleted, []string{"feature/old"}) {
		t.Fatalf("deleted = %#v, want %#v", client.deleted, []string{"feature/old"})
	}
}

func TestRunRemoveCancelsWithoutConfirmation(t *testing.T) {
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main", Main: true},
			{Path: "/repo-feature", Branch: "feature/one"},
		},
	}
	stdout := &bytes.Buffer{}
	app := New(client, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"remove", "--regex", `.*`}, strings.NewReader("n\n"), stdout, io.Discard)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	assertContains(t, stdout.String(), "grape remove will delete:")
	assertContains(t, stdout.String(), "/repo-feature")
	assertContains(t, stdout.String(), "grape: removal cancelled")
	if len(client.removed) != 0 {
		t.Fatalf("removed = %#v, want none", client.removed)
	}
	if len(client.deleted) != 0 {
		t.Fatalf("deleted = %#v, want none", client.deleted)
	}
}

func TestRunRemoveReturnsTwoForInvalidRegex(t *testing.T) {
	stderr := &bytes.Buffer{}
	app := New(&fakeClient{}, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"remove", "--regex", "["}, nil, io.Discard, stderr)

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	assertContains(t, stderr.String(), "error parsing regexp")
}

func TestRunResetRemovesNonDefaultWorktreesAndBranchesThenAddsConfiguredWorktrees(t *testing.T) {
	configData := []byte(`{
		"default_branch": "main",
		"worktrees": [
			{"path": "../repo-feature-one", "branch": "feature/one"},
			{"path": "../repo-feature-two", "branch": "feature/two", "start_point": "origin/develop"}
		]
	}`)
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "feature/current", Main: true},
			{Path: "/repo-feature", Branch: "feature/old"},
			{Path: "/repo-detached"},
		},
		branches: []string{"main", "feature/current", "feature/old", "feature/orphan"},
	}
	app := New(client, &fakeRunner{}, func(path string) ([]byte, error) {
		if path != "grape.json" {
			t.Fatalf("path = %q, want grape.json", path)
		}
		return configData, nil
	})

	stdout := &bytes.Buffer{}
	code := app.Run(context.Background(), []string{"reset", "--config", "grape.json"}, strings.NewReader("y\n"), stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	assertContains(t, stdout.String(), "grape reset will delete:")
	assertContains(t, stdout.String(), "/repo-feature")
	assertContains(t, stdout.String(), "feature/orphan")
	if !slices.Equal(client.removed, []string{"/repo-feature", "/repo-detached"}) {
		t.Fatalf("removed = %#v, want %#v", client.removed, []string{"/repo-feature", "/repo-detached"})
	}
	if !slices.Equal(client.deleted, []string{"feature/old", "feature/orphan"}) {
		t.Fatalf("deleted = %#v, want %#v", client.deleted, []string{"feature/old", "feature/orphan"})
	}
	wantAdded := []worktree.ConfiguredItem{
		{Path: "../repo-feature-one", Branch: "feature/one"},
		{Path: "../repo-feature-two", Branch: "feature/two", StartPoint: "origin/develop"},
	}
	if !slices.Equal(client.added, wantAdded) {
		t.Fatalf("added = %#v, want %#v", client.added, wantAdded)
	}
	if !slices.Equal(client.addedDefaults, []string{"main", "main"}) {
		t.Fatalf("addedDefaults = %#v, want %#v", client.addedDefaults, []string{"main", "main"})
	}
}

func TestRunResetCancelsWhenConfirmationIsNotYes(t *testing.T) {
	configData := []byte(`{
		"default_branch": "main",
		"worktrees": [
			{"path": "../repo-feature-one", "branch": "feature/one"}
		]
	}`)
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main", Main: true},
			{Path: "/repo-feature", Branch: "feature/old"},
		},
		branches: []string{"main", "feature/old"},
	}
	stdout := &bytes.Buffer{}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) {
		return configData, nil
	})

	code := app.Run(context.Background(), []string{"reset", "--config", "grape.json"}, strings.NewReader("n\n"), stdout, io.Discard)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	assertContains(t, stdout.String(), "grape reset will delete:")
	assertContains(t, stdout.String(), "grape: reset cancelled")
	if len(client.removed) != 0 {
		t.Fatalf("removed = %#v, want none", client.removed)
	}
	if len(client.deleted) != 0 {
		t.Fatalf("deleted = %#v, want none", client.deleted)
	}
	if len(client.added) != 0 {
		t.Fatalf("added = %#v, want none", client.added)
	}
}

func TestRunResetDetectsDefaultBranchWhenConfigOmitsIt(t *testing.T) {
	configData := []byte(`{"worktrees":[{"path":"../repo-feature","branch":"feature/example"}]}`)
	client := &fakeClient{
		defaultBranch: "main",
		branches:      []string{"main"},
	}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) {
		return configData, nil
	}).WithPathResolver(config.PathResolver{
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		Env: func(key string) string {
			if key == "GRAPE_HOME" {
				return "/env/grape"
			}
			return ""
		},
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	})

	code := app.Run(context.Background(), []string{"reset"}, strings.NewReader("y\n"), io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !slices.Equal(client.addedDefaults, []string{"main"}) {
		t.Fatalf("addedDefaults = %#v, want %#v", client.addedDefaults, []string{"main"})
	}
}

func TestRunResetReadsGrapeHomeConfigWhenCurrentConfigIsMissing(t *testing.T) {
	configData := []byte(`{"worktrees":[{"path":"../repo-feature","branch":"feature/example"}]}`)
	client := &fakeClient{
		defaultBranch: "main",
		branches:      []string{"main"},
	}
	app := New(client, &fakeRunner{}, func(path string) ([]byte, error) {
		want := filepath.Join("/env/grape", "grape.json")
		if path != want {
			t.Fatalf("path = %q, want %q", path, want)
		}
		return configData, nil
	}).WithPathResolver(config.PathResolver{
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		Env: func(key string) string {
			if key == "GRAPE_HOME" {
				return "/env/grape"
			}
			return ""
		},
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	})

	code := app.Run(context.Background(), []string{"reset"}, strings.NewReader("y\n"), io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}
