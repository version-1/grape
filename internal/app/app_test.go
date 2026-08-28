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
	"sync"
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
	mu              sync.Mutex
	worktrees       []worktree.Worktree
	branches        []string
	branchListErr   error
	branchLists     int
	defaultBranch   string
	listErr         error
	removeErr       error
	deleteErr       error
	addErr          error
	removed         []string
	removedForce    []bool
	deleted         []string
	added           []worktree.ConfiguredItem
	addedDefaults   []string
	currentBranch   string
	rebaseBranch    string
	rebaseBranchErr error
	originURL       string
	validateErr     error
	pushErr         error
	pushedBranch    string
	pushedForce     bool
	rebaseErr       error
	rebaseArgs      []string
	rebaseStdin     io.Reader
	rebaseStdout    io.Writer
	rebaseStderr    io.Writer
	gitCalls        []string
	removeHook      func(string)
	addHook         func(worktree.ConfiguredItem)
	deleteHook      func([]string)
}

func (c *fakeClient) CurrentBranch(context.Context) (string, error) {
	c.gitCalls = append(c.gitCalls, "current-branch")
	if c.listErr != nil {
		return "", c.listErr
	}
	return c.currentBranch, nil
}

func (c *fakeClient) RebaseBranch(context.Context) (string, error) {
	c.gitCalls = append(c.gitCalls, "rebase-branch")
	if c.rebaseBranchErr != nil {
		return "", c.rebaseBranchErr
	}
	return c.rebaseBranch, nil
}

func (c *fakeClient) ValidateBranch(_ context.Context, branch string) error {
	c.gitCalls = append(c.gitCalls, "validate:"+branch)
	return c.validateErr
}

func (c *fakeClient) OriginURL(context.Context) (string, error) {
	c.gitCalls = append(c.gitCalls, "origin-url")
	if c.listErr != nil {
		return "", c.listErr
	}
	return c.originURL, nil
}

func (c *fakeClient) Push(_ context.Context, branch string, force bool, _ io.Writer, _ io.Writer) error {
	c.gitCalls = append(c.gitCalls, "push")
	c.pushedBranch = branch
	c.pushedForce = force
	return c.pushErr
}

func (c *fakeClient) Rebase(_ context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	c.gitCalls = append(c.gitCalls, "rebase")
	c.rebaseArgs = append([]string(nil), args...)
	c.rebaseStdin = stdin
	c.rebaseStdout = stdout
	c.rebaseStderr = stderr
	return c.rebaseErr
}

func (c *fakeClient) ListWorktrees(context.Context) ([]worktree.Worktree, error) {
	return c.worktrees, c.listErr
}

func (c *fakeClient) RemoveWorktree(_ context.Context, path string, force bool, _ io.Writer, _ io.Writer) error {
	if c.removeHook != nil {
		c.removeHook(path)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removed = append(c.removed, path)
	c.removedForce = append(c.removedForce, force)
	return c.removeErr
}

func (c *fakeClient) DeleteBranch(_ context.Context, branch string, _ io.Writer, _ io.Writer) error {
	return c.DeleteBranches(context.Background(), []string{branch}, io.Discard, io.Discard)
}

func (c *fakeClient) DeleteBranches(_ context.Context, branches []string, _ io.Writer, _ io.Writer) error {
	if c.deleteHook != nil {
		c.deleteHook(branches)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, branch := range branches {
		c.deleted = append(c.deleted, branch)
		c.branches = slices.DeleteFunc(c.branches, func(candidate string) bool { return candidate == branch })
	}
	return c.deleteErr
}

func (c *fakeClient) ListBranches(context.Context) ([]string, error) {
	c.branchLists++
	if c.branchLists > 1 && c.branchListErr != nil {
		return nil, c.branchListErr
	}
	return c.branches, c.listErr
}

func (c *fakeClient) DefaultBranch(context.Context) (string, error) {
	return c.defaultBranch, c.listErr
}

func (c *fakeClient) AddWorktree(_ context.Context, item worktree.ConfiguredItem, defaultBranch string, _ io.Writer, _ io.Writer) error {
	if c.addHook != nil {
		c.addHook(item)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.added = append(c.added, item)
	c.addedDefaults = append(c.addedDefaults, defaultBranch)
	if !slices.Contains(c.branches, item.Branch) {
		c.branches = append(c.branches, item.Branch)
		slices.Sort(c.branches)
	}
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
	assertContains(t, stdout.String(), "grape init")
	assertContains(t, stdout.String(), "grape reset [--yes|-y] [--config|-c <path>]")
	assertContains(t, stdout.String(), "grape rebase [--config|-c <path>] [--] [<git-rebase-args>...]")
	assertContains(t, stdout.String(), "$GRAPE_HOME/grape.json")
	assertContains(t, stdout.String(), "except rebase, which requires an existing config file")
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

func TestRunInitCreatesConfigInGrapeHome(t *testing.T) {
	temporaryDir := t.TempDir()
	examplePath := filepath.Join(temporaryDir, "grape.example.json")
	want := []byte(`{"worktrees":[]}`)
	if err := os.WriteFile(examplePath, want, 0o600); err != nil {
		t.Fatalf("write example: %v", err)
	}
	stdout := &bytes.Buffer{}
	app := New(&fakeClient{}, &fakeRunner{}, nil).
		WithExampleConfigPath(examplePath).
		WithPathResolver(config.PathResolver{
			Env:      func(string) string { return "" },
			UserHome: func() (string, error) { return temporaryDir, nil },
		})

	code := app.Run(context.Background(), []string{"init"}, nil, stdout, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	destination := filepath.Join(temporaryDir, ".grape", "grape.json")
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("config = %q, want %q", got, want)
	}
	assertContains(t, stdout.String(), destination)
}

func TestRunInitDoesNotOverwriteExistingConfig(t *testing.T) {
	temporaryDir := t.TempDir()
	examplePath := filepath.Join(temporaryDir, "grape.example.json")
	if err := os.WriteFile(examplePath, []byte(`{"worktrees":["example"]}`), 0o600); err != nil {
		t.Fatalf("write example: %v", err)
	}
	destination := filepath.Join(temporaryDir, ".grape", "grape.json")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	want := []byte(`{"worktrees":["existing"]}`)
	if err := os.WriteFile(destination, want, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	stderr := &bytes.Buffer{}
	app := New(&fakeClient{}, &fakeRunner{}, nil).
		WithExampleConfigPath(examplePath).
		WithPathResolver(config.PathResolver{
			Env:      func(string) string { return "" },
			UserHome: func() (string, error) { return temporaryDir, nil },
		})

	code := app.Run(context.Background(), []string{"init"}, nil, io.Discard, stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("config = %q, want existing value %q", got, want)
	}
	assertContains(t, stderr.String(), "config already exists")
}

func TestRunInitRejectsArguments(t *testing.T) {
	stderr := &bytes.Buffer{}
	app := New(&fakeClient{}, &fakeRunner{}, nil)

	code := app.Run(context.Background(), []string{"init", "extra"}, nil, io.Discard, stderr)

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if got, want := stderr.String(), "grape: usage: grape init\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
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
	if !slices.Equal(client.removedForce, []bool{false, false}) {
		t.Fatalf("removed force = %#v, want remove to remain non-forced", client.removedForce)
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
	assertContains(t, stdout.String(), "Branches\n  feature/current\n  feature/one\n  feature/two\n  main\n")
	for _, unwanted := range []string{"Removing worktree", "Deleting branch", "Adding worktree"} {
		if strings.Contains(stdout.String(), unwanted) {
			t.Fatalf("stdout contains progress log %q: %q", unwanted, stdout)
		}
	}
	slices.Sort(client.removed)
	if !slices.Equal(client.removed, []string{"/repo-detached", "/repo-feature"}) {
		t.Fatalf("removed = %#v, want %#v", client.removed, []string{"/repo-feature", "/repo-detached"})
	}
	if !slices.Equal(client.removedForce, []bool{true, true}) {
		t.Fatalf("removed force = %#v, want %#v", client.removedForce, []bool{true, true})
	}
	if !slices.Equal(client.deleted, []string{"feature/old", "feature/orphan"}) {
		t.Fatalf("deleted = %#v, want %#v", client.deleted, []string{"feature/old", "feature/orphan"})
	}
	wantAdded := []worktree.ConfiguredItem{
		{Path: "../repo-feature-one", Branch: "feature/one"},
		{Path: "../repo-feature-two", Branch: "feature/two", StartPoint: "origin/develop"},
	}
	slices.SortFunc(client.added, func(left, right worktree.ConfiguredItem) int {
		return strings.Compare(left.Path, right.Path)
	})
	if !slices.Equal(client.added, wantAdded) {
		t.Fatalf("added = %#v, want %#v", client.added, wantAdded)
	}
	if !slices.Equal(client.addedDefaults, []string{"main", "main"}) {
		t.Fatalf("addedDefaults = %#v, want %#v", client.addedDefaults, []string{"main", "main"})
	}
}

func TestRunResetBatchesBranchDeletion(t *testing.T) {
	configData := []byte(`{"default_branch":"main","worktrees":[{"path":"../repo-new","branch":"feature/new"}]}`)
	client := &fakeClient{
		worktrees: []worktree.Worktree{{Path: "/repo", Branch: "main", Main: true}},
		branches:  []string{"feature/one", "feature/two", "main"},
	}
	var deleteCalls [][]string
	client.deleteHook = func(branches []string) {
		deleteCalls = append(deleteCalls, append([]string(nil), branches...))
	}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) { return configData, nil })

	code := app.Run(context.Background(), []string{"reset", "-y", "--config", "grape.json"}, nil, io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if len(deleteCalls) != 1 || !slices.Equal(deleteCalls[0], []string{"feature/one", "feature/two"}) {
		t.Fatalf("delete calls = %#v, want one batch", deleteCalls)
	}
}

func TestRunResetWaitsForWorktreeRemovalBeforeDeletingBranchesAndAddsSerially(t *testing.T) {
	configData := []byte(`{"default_branch":"main","worktrees":[{"path":"../repo-new-one","branch":"feature/new-one"},{"path":"../repo-new-two","branch":"feature/new-two"}]}`)
	removalsStarted := make(chan string, 2)
	allowRemovals := make(chan struct{})
	branchesDeleted := make(chan []string, 1)
	additionsStarted := make(chan string, 2)
	allowAdditions := make(chan struct{})
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main", Main: true},
			{Path: "/repo-one", Branch: "feature/one"},
			{Path: "/repo-two", Branch: "feature/two"},
		},
		branches: []string{"feature/one", "feature/two", "main"},
	}
	client.removeHook = func(path string) {
		removalsStarted <- path
		<-allowRemovals
	}
	client.deleteHook = func(branches []string) { branchesDeleted <- branches }
	client.addHook = func(item worktree.ConfiguredItem) {
		additionsStarted <- item.Path
		<-allowAdditions
	}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) { return configData, nil })
	result := make(chan int, 1)
	go func() {
		result <- app.Run(context.Background(), []string{"reset", "-y", "--config", "grape.json"}, nil, io.Discard, io.Discard)
	}()

	<-removalsStarted
	<-removalsStarted
	select {
	case branches := <-branchesDeleted:
		t.Fatalf("branches deleted before removals completed: %#v", branches)
	default:
	}
	allowRemovals <- struct{}{}
	allowRemovals <- struct{}{}

	select {
	case branches := <-branchesDeleted:
		if !slices.Equal(branches, []string{"feature/one", "feature/two"}) {
			t.Fatalf("deleted branches = %#v", branches)
		}
	case <-result:
		t.Fatal("reset completed before deleting branches")
	}
	<-additionsStarted
	select {
	case path := <-additionsStarted:
		t.Fatalf("addition started before the previous addition completed: %s", path)
	default:
	}
	allowAdditions <- struct{}{}
	<-additionsStarted
	allowAdditions <- struct{}{}
	if code := <-result; code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
}

func TestRunResetStopsAfterFailingRemovalPair(t *testing.T) {
	configData := []byte(`{"default_branch":"main","worktrees":[{"path":"../repo-new","branch":"feature/new"}]}`)
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main", Main: true},
			{Path: "/repo-one", Branch: "feature/one"},
			{Path: "/repo-two", Branch: "feature/two"},
			{Path: "/repo-three", Branch: "feature/three"},
		},
		branches:  []string{"feature/one", "feature/two", "feature/three", "main"},
		removeErr: errors.New("remove failed"),
	}
	stderr := &bytes.Buffer{}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) { return configData, nil })

	code := app.Run(context.Background(), []string{"reset", "-y", "--config", "grape.json"}, nil, io.Discard, stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(client.removed) != 3 {
		t.Fatalf("removed = %#v, want all initially started removals", client.removed)
	}
	if len(client.deleted) != 0 || len(client.added) != 0 {
		t.Fatalf("continued after removal failure: deleted=%#v added=%#v", client.deleted, client.added)
	}
	assertContains(t, stderr.String(), "remove worktree /repo-one: remove failed")
}

func TestRunResetStopsAfterFailingAdditions(t *testing.T) {
	configData := []byte(`{"default_branch":"main","worktrees":[{"path":"../repo-one","branch":"feature/one"},{"path":"../repo-two","branch":"feature/two"},{"path":"../repo-three","branch":"feature/three"}]}`)
	client := &fakeClient{
		worktrees: []worktree.Worktree{{Path: "/repo", Branch: "main", Main: true}},
		branches:  []string{"main"},
		addErr:    errors.New("add failed"),
	}
	stderr := &bytes.Buffer{}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) { return configData, nil })

	code := app.Run(context.Background(), []string{"reset", "-y", "--config", "grape.json"}, nil, io.Discard, stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(client.added) != 1 {
		t.Fatalf("added = %#v, want only the failed serial addition", client.added)
	}
	if client.branchLists != 1 {
		t.Fatalf("branch lists = %d, want no final branch list", client.branchLists)
	}
	assertContains(t, stderr.String(), "add worktree ../repo-one: add failed")
}

func TestRunWithConcurrencyLimitKeepsFiveRemovalsActive(t *testing.T) {
	started := make(chan int, 6)
	releases := make([]chan struct{}, 6)
	for index := range releases {
		releases[index] = make(chan struct{})
	}
	result := make(chan error, 1)

	go func() {
		_, err := runWithConcurrencyLimit([]int{0, 1, 2, 3, 4, 5}, resetRemoveConcurrency, func(item int, _ io.Writer, _ io.Writer) error {
			started <- item
			<-releases[item]
			return nil
		}, io.Discard, io.Discard)
		result <- err
	}()

	initial := map[int]struct{}{}
	for range resetRemoveConcurrency {
		item := <-started
		if _, exists := initial[item]; exists {
			t.Fatalf("operation %d started twice", item)
		}
		initial[item] = struct{}{}
	}
	select {
	case item := <-started:
		t.Fatalf("operation %d started before one of five active operations completed", item)
	default:
	}
	releases[0] <- struct{}{}
	if item := <-started; item != 5 {
		t.Fatalf("started item = %d, want 5 after a slot was freed", item)
	}
	for index := 1; index < len(releases); index++ {
		releases[index] <- struct{}{}
	}
	if err := <-result; err != nil {
		t.Fatalf("runWithConcurrencyLimit() error = %v", err)
	}
}

func TestRunResetYesSkipsConfirmation(t *testing.T) {
	configData := []byte(`{"default_branch":"main","worktrees":[{"path":"../repo-feature","branch":"feature/new"}]}`)
	client := &fakeClient{
		worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main", Main: true},
			{Path: "/repo-old", Branch: "feature/old"},
		},
		branches: []string{"feature/old", "main"},
	}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) { return configData, nil })

	for _, yesFlag := range []string{"-y", "--yes"} {
		t.Run(yesFlag, func(t *testing.T) {
			client.removed = nil
			client.removedForce = nil
			client.deleted = nil
			client.added = nil
			client.branches = []string{"feature/old", "main"}
			client.branchLists = 0
			stdout := &bytes.Buffer{}

			code := app.Run(context.Background(), []string{"reset", "--config", "grape.json", yesFlag}, nil, stdout, io.Discard)

			if code != 0 {
				t.Fatalf("code = %d, want 0", code)
			}
			if strings.Contains(stdout.String(), "Proceed with reset?") {
				t.Fatalf("stdout contains confirmation prompt: %q", stdout)
			}
			assertContains(t, stdout.String(), "grape reset will delete:")
			assertContains(t, stdout.String(), "Branches\n  feature/new\n  main\n")
		})
	}
}

func TestRunResetRejectsForceFlags(t *testing.T) {
	for _, forceFlag := range []string{"-f", "--force"} {
		t.Run(forceFlag, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			client := &fakeClient{}

			code := New(client, &fakeRunner{}, nil).Run(context.Background(), []string{"reset", forceFlag}, nil, io.Discard, stderr)

			if code != 2 {
				t.Fatalf("code = %d, want 2", code)
			}
			if client.branchLists != 0 || len(client.removed) != 0 {
				t.Fatalf("client was called: %#v", client)
			}
		})
	}
}

func TestRunResetFailsWhenFinalBranchListFails(t *testing.T) {
	configData := []byte(`{"default_branch":"main","worktrees":[{"path":"../repo-feature","branch":"feature/new"}]}`)
	client := &fakeClient{
		worktrees:     []worktree.Worktree{{Path: "/repo", Branch: "main", Main: true}},
		branches:      []string{"main"},
		branchListErr: errors.New("final list failed"),
	}
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	app := New(client, &fakeRunner{}, func(string) ([]byte, error) { return configData, nil })

	code := app.Run(context.Background(), []string{"reset", "-y", "-c", "grape.json"}, nil, stdout, stderr)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	assertContains(t, stderr.String(), "list branches: final list failed")
	if strings.Contains(stdout.String(), "Branches\n  main") {
		t.Fatalf("stdout contains final branch list: %q", stdout)
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
		Stat: func(path string) (os.FileInfo, error) {
			if path == filepath.Join("/env/grape", "grape.json") {
				return nil, nil
			}
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
		Stat: func(path string) (os.FileInfo, error) {
			if path == filepath.Join("/env/grape", "grape.json") {
				return nil, nil
			}
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

func TestRunPushAcceptsOnlyDocumentedForms(t *testing.T) {
	valid := []struct {
		name  string
		args  []string
		force bool
	}{
		{"normal", []string{"push"}, false},
		{"force with lease", []string{"push", "--force-with-lease"}, true},
	}
	for _, test := range valid {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeClient{currentBranch: "feature/test", originURL: "ssh://origin"}
			stderr := &bytes.Buffer{}
			application := withConfig(New(client, &fakeRunner{}, nil), `{}`)
			code := application.Run(context.Background(), test.args, nil, io.Discard, stderr)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr)
			}
			if client.pushedBranch != "feature/test" || client.pushedForce != test.force {
				t.Fatalf("push = %q force=%t", client.pushedBranch, client.pushedForce)
			}
			assertContains(t, stderr.String(), "grape: branch: feature/test")
			assertContains(t, stderr.String(), "grape: origin: ssh://origin")
		})
	}

	invalid := [][]string{
		{"push", "--force"},
		{"push", "-f"},
		{"push", "--force-with-lease=main:abc"},
		{"push", "origin"},
		{"push", "HEAD:other"},
		{"push", "--tags"},
		{"push", "--force-with-lease", "extra"},
	}
	for _, args := range invalid {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			client := &fakeClient{currentBranch: "feature/test", originURL: "ssh://origin"}
			code := withConfig(New(client, &fakeRunner{}, nil), `{}`).
				Run(context.Background(), args, nil, io.Discard, io.Discard)
			if code != 2 {
				t.Fatalf("code = %d, want 2", code)
			}
			if len(client.gitCalls) != 0 {
				t.Fatalf("git calls = %#v, want none", client.gitCalls)
			}
		})
	}
}

func TestRunRebasePassesGitArgumentsAndPreservesStreams(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		gitArgs []string
	}{
		{"tracking upstream", []string{"rebase"}, nil},
		{"interactive upstream", []string{"rebase", "-i", "origin/main"}, []string{"-i", "origin/main"}},
		{"onto with multiple revisions", []string{"rebase", "--onto", "main", "base", "feature"}, []string{"--onto", "main", "base", "feature"}},
		{"continue", []string{"rebase", "--continue"}, []string{"--continue"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeClient{currentBranch: "feature/test"}
			stdin := strings.NewReader("input")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			code := withConfig(New(client, &fakeRunner{}, nil), `{"rebase":{"allowed_branches":["feature/*"]}}`).
				Run(context.Background(), test.args, stdin, stdout, stderr)

			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr)
			}
			if !slices.Equal(client.rebaseArgs, test.gitArgs) {
				t.Fatalf("git args = %#v, want %#v", client.rebaseArgs, test.gitArgs)
			}
			if client.rebaseStdin != stdin || client.rebaseStdout != stdout || client.rebaseStderr != stderr {
				t.Fatal("rebase streams were not preserved")
			}
			if !slices.Equal(client.gitCalls, []string{"current-branch", "rebase"}) {
				t.Fatalf("git calls = %#v", client.gitCalls)
			}
		})
	}
}

func TestRunRebaseRejectsMissingGrapeConfigPathBeforeGit(t *testing.T) {
	tests := [][]string{
		{"rebase", "--config"},
		{"rebase", "--config="},
		{"rebase", "--config", ""},
		{"rebase", "-c", ""},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			client := &fakeClient{currentBranch: "feature/test"}
			code := withConfig(New(client, &fakeRunner{}, nil), `{"rebase":{"allowed_branches":["feature/*"]}}`).
				Run(context.Background(), args, nil, io.Discard, io.Discard)
			if code != 2 {
				t.Fatalf("code = %d, want 2", code)
			}
			if len(client.gitCalls) != 0 {
				t.Fatalf("git calls = %#v, want none", client.gitCalls)
			}
		})
	}
}

func TestRunRebaseRejectsPolicyAndDetachedHEADBeforeRebase(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		client     *fakeClient
		wantCode   int
		wantStderr string
		wantCalls  []string
	}{
		{"missing rebase", `{}`, &fakeClient{currentBranch: "feature/test"}, 1, "feature/test", []string{"current-branch"}},
		{"missing allowed branches", `{"rebase":{}}`, &fakeClient{currentBranch: "feature/test"}, 1, "feature/test", []string{"current-branch"}},
		{"empty policy", `{"rebase":{"allowed_branches":[]}}`, &fakeClient{currentBranch: "feature/test"}, 1, "feature/test", []string{"current-branch"}},
		{"denied branch", `{"rebase":{"allowed_branches":["feature/*"]}}`, &fakeClient{currentBranch: "fix/test"}, 1, "fix/test", []string{"current-branch"}},
		{"detached head", `{"rebase":{"allowed_branches":["feature/*"]}}`, &fakeClient{listErr: errors.New("detached"), rebaseBranchErr: errors.New("no rebase")}, 1, "detached", []string{"current-branch", "rebase-branch"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			code := withConfig(New(test.client, &fakeRunner{}, nil), test.config).
				Run(context.Background(), []string{"rebase"}, nil, io.Discard, stderr)
			if code != test.wantCode {
				t.Fatalf("code = %d, want %d", code, test.wantCode)
			}
			assertContains(t, stderr.String(), test.wantStderr)
			if !slices.Equal(test.client.gitCalls, test.wantCalls) {
				t.Fatalf("git calls = %#v, want %#v", test.client.gitCalls, test.wantCalls)
			}
		})
	}
}

func TestRunRebaseUsesOriginalBranchDuringActiveRebase(t *testing.T) {
	client := &fakeClient{listErr: errors.New("detached"), rebaseBranch: "feature/test"}
	code := withConfig(New(client, &fakeRunner{}, nil), `{"rebase":{"allowed_branches":["feature/*"]}}`).
		Run(context.Background(), []string{"rebase", "--continue"}, nil, io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !slices.Equal(client.gitCalls, []string{"current-branch", "rebase-branch", "rebase"}) {
		t.Fatalf("git calls = %#v", client.gitCalls)
	}
	if !slices.Equal(client.rebaseArgs, []string{"--continue"}) {
		t.Fatalf("git args = %#v", client.rebaseArgs)
	}
}

func TestRunRebaseExtractsGrapeConfigUntilSeparator(t *testing.T) {
	client := &fakeClient{currentBranch: "worktrees/3"}
	application := New(client, &fakeRunner{}, func(path string) ([]byte, error) {
		if path != "/tmp/rebase.json" {
			t.Fatalf("path = %q, want explicit config", path)
		}
		return []byte(`{"rebase":{"allowed_branches":["worktrees/*"]}}`), nil
	})

	code := application.Run(context.Background(), []string{
		"rebase", "-i", "--config=/tmp/first.json", "-c", "/tmp/rebase.json", "origin/main", "--", "--config", "git-value",
	}, nil, io.Discard, io.Discard)

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	want := []string{"-i", "origin/main", "--config", "git-value"}
	if !slices.Equal(client.rebaseArgs, want) {
		t.Fatalf("git args = %#v, want %#v", client.rebaseArgs, want)
	}
}

func TestRunRebaseRequiresValidExistingConfigBeforeGit(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{"malformed JSON", `{"rebase":`},
		{"invalid type", `{"rebase":{"allowed_branches":"feature/*"}}`},
		{"invalid later glob", `{"rebase":{"allowed_branches":["feature/*","["]}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeClient{currentBranch: "feature/test"}
			code := withConfig(New(client, &fakeRunner{}, nil), test.config).
				Run(context.Background(), []string{"rebase"}, nil, io.Discard, io.Discard)
			if code != 2 {
				t.Fatalf("code = %d, want 2", code)
			}
			if len(client.gitCalls) != 0 {
				t.Fatalf("git calls = %#v, want none", client.gitCalls)
			}
		})
	}

	client := &fakeClient{currentBranch: "feature/test"}
	code := New(client, &fakeRunner{}, nil).
		Run(context.Background(), []string{"rebase"}, nil, io.Discard, io.Discard)
	if code != 2 {
		t.Fatalf("missing config code = %d, want 2", code)
	}
	if len(client.gitCalls) != 0 {
		t.Fatalf("missing config git calls = %#v, want none", client.gitCalls)
	}
}

func TestRunRebaseUsesExplicitConfigAndPreservesGitExitCode(t *testing.T) {
	client := &fakeClient{
		currentBranch: "worktrees/3",
		rebaseErr:     fakeExitError{code: 23},
	}
	application := New(client, &fakeRunner{}, func(path string) ([]byte, error) {
		if path != "/tmp/rebase.json" {
			t.Fatalf("path = %q, want explicit config", path)
		}
		return []byte(`{"rebase":{"allowed_branches":["worktrees/*"]}}`), nil
	})

	code := application.Run(
		context.Background(),
		[]string{"rebase", "--config", "/tmp/rebase.json", "origin/main"},
		nil,
		io.Discard,
		io.Discard,
	)

	if code != 23 {
		t.Fatalf("code = %d, want 23", code)
	}
	if !slices.Equal(client.rebaseArgs, []string{"origin/main"}) {
		t.Fatalf("git args = %#v, want origin/main", client.rebaseArgs)
	}
}

func TestRunPushRefusesProtectedBranchesInBothModes(t *testing.T) {
	tests := []struct {
		name   string
		config string
		branch string
	}{
		{"default", `{}`, "main"},
		{"configured exact", `{"push":{"protected_branches":["trunk"]}}`, "trunk"},
		{"configured glob", `{"push":{"protected_branches":["release/*"]}}`, "release/1.0"},
	}
	for _, test := range tests {
		for _, forceArg := range [][]string{{"push"}, {"push", "--force-with-lease"}} {
			t.Run(test.name+"/"+strings.Join(forceArg, " "), func(t *testing.T) {
				client := &fakeClient{currentBranch: test.branch, originURL: "ssh://origin"}
				stderr := &bytes.Buffer{}
				code := withConfig(New(client, &fakeRunner{}, nil), test.config).
					Run(context.Background(), forceArg, nil, io.Discard, stderr)
				if code != 1 {
					t.Fatalf("code = %d, want 1", code)
				}
				assertContains(t, stderr.String(), "refusing to push protected branch")
				if client.pushedBranch != "" {
					t.Fatalf("pushed branch = %q", client.pushedBranch)
				}
			})
		}
	}
}

func TestRunPushPreservesGitExitCode(t *testing.T) {
	client := &fakeClient{
		currentBranch: "feature/test",
		originURL:     "ssh://origin",
		pushErr:       fakeExitError{code: 23},
	}
	code := withConfig(New(client, &fakeRunner{}, nil), `{}`).
		Run(context.Background(), []string{"push"}, nil, io.Discard, io.Discard)
	if code != 23 {
		t.Fatalf("code = %d, want 23", code)
	}
}

func TestRunPushStopsBeforeGitForInvalidConfig(t *testing.T) {
	client := &fakeClient{currentBranch: "feature/test", originURL: "ssh://origin"}
	code := withConfig(New(client, &fakeRunner{}, nil), `{"unknown":true}`).
		Run(context.Background(), []string{"push"}, nil, io.Discard, io.Discard)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if len(client.gitCalls) != 0 {
		t.Fatalf("git calls = %#v, want none", client.gitCalls)
	}
}

func TestRunPushValidatesConfigBeforeArguments(t *testing.T) {
	stderr := &bytes.Buffer{}
	code := withConfig(New(&fakeClient{}, &fakeRunner{}, nil), `{"unknown":true}`).
		Run(context.Background(), []string{"push", "--force"}, nil, io.Discard, stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	assertContains(t, stderr.String(), "unknown field")
	if strings.Contains(stderr.String(), "usage: grape push") {
		t.Fatalf("arguments were validated before config: %q", stderr)
	}
}

func TestRunPushRejectsSymbolicAndBranchValidationFailures(t *testing.T) {
	tests := []struct {
		name       string
		client     *fakeClient
		want       string
		wantCalls  []string
		wantStatus int
	}{
		{
			name:       "symbolic branch failure",
			client:     &fakeClient{listErr: errors.New("not a repository")},
			want:       "current HEAD is detached or not in a git repository",
			wantCalls:  []string{"current-branch"},
			wantStatus: 1,
		},
		{
			name:       "branch begins with dash",
			client:     &fakeClient{currentBranch: "-danger", validateErr: errors.New("invalid")},
			want:       "refusing to push branch that starts with '-'",
			wantCalls:  []string{"current-branch", "validate:-danger"},
			wantStatus: 1,
		},
		{
			name:       "invalid branch",
			client:     &fakeClient{currentBranch: "invalid name", validateErr: errors.New("invalid")},
			want:       "invalid branch name",
			wantCalls:  []string{"current-branch", "validate:invalid name"},
			wantStatus: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			code := withConfig(New(test.client, &fakeRunner{}, nil), `{}`).
				Run(context.Background(), []string{"push"}, nil, io.Discard, stderr)
			if code != test.wantStatus {
				t.Fatalf("code = %d, want %d", code, test.wantStatus)
			}
			assertContains(t, stderr.String(), test.want)
			if !slices.Equal(test.client.gitCalls, test.wantCalls) {
				t.Fatalf("calls = %#v, want %#v", test.client.gitCalls, test.wantCalls)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("input failed")
}

func TestConfirmationInputErrorsUseStderr(t *testing.T) {
	client := &fakeClient{worktrees: []worktree.Worktree{{Path: "/repo-feature", Branch: "feature/test"}}}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := withConfig(New(client, &fakeRunner{}, nil), `{}`).
		Run(context.Background(), []string{"remove", "/repo-feature"}, failingReader{}, stdout, stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	assertContains(t, stderr.String(), "grape: read confirmation: input failed")
	if strings.Contains(stdout.String(), "read confirmation") {
		t.Fatalf("confirmation error written to stdout: %q", stdout)
	}
}

func TestRunPushWarnsAndContinuesWhenConfigIsMissing(t *testing.T) {
	client := &fakeClient{currentBranch: "feature/test", originURL: "ssh://origin"}
	stderr := &bytes.Buffer{}
	code := New(client, &fakeRunner{}, nil).
		Run(context.Background(), []string{"push"}, nil, io.Discard, stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	assertContains(t, stderr.String(), "grape: warning: grape.json not found; using default protected branches: main, master")
}

func TestGlobalConfigValidationAppliesToBuiltIns(t *testing.T) {
	tests := [][]string{
		{"list"},
		{"branch", "feature/test"},
		{"remove", "prefix"},
		{"reset"},
		{"push"},
		{"rebase"},
	}
	for _, args := range tests {
		t.Run(args[0], func(t *testing.T) {
			client := &fakeClient{}
			code := withConfig(New(client, &fakeRunner{}, nil), `{"unknown":true}`).
				Run(context.Background(), args, nil, io.Discard, io.Discard)
			if code != 2 {
				t.Fatalf("code = %d, want 2", code)
			}
			if len(client.gitCalls) != 0 || len(client.removed) != 0 {
				t.Fatalf("client was called: %#v", client)
			}
		})
	}
}

func TestGlobalConfigValidationExcludesHelpVersionInitAndDelegation(t *testing.T) {
	tests := [][]string{
		{"help"},
		{"version"},
		{"init", "extra"},
		{"list", "--porcelain"},
		{"prune"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			reads := 0
			runner := &fakeRunner{}
			application := New(&fakeClient{}, runner, func(string) ([]byte, error) {
				reads++
				return []byte(`{"unknown":true}`), nil
			}).WithPathResolver(config.PathResolver{
				Stat: func(string) (os.FileInfo, error) { return nil, nil },
				Env:  func(string) string { return "" },
				UserHome: func() (string, error) {
					return "/home/user", nil
				},
			})
			_ = application.Run(context.Background(), args, nil, io.Discard, io.Discard)
			if reads != 0 {
				t.Fatalf("config reads = %d, want 0", reads)
			}
		})
	}
}

func withConfig(application App, contents string) App {
	application.readFile = func(string) ([]byte, error) {
		return []byte(contents), nil
	}
	return application.WithPathResolver(config.PathResolver{
		Stat: func(path string) (os.FileInfo, error) {
			if path == "grape.json" {
				return nil, nil
			}
			return nil, os.ErrNotExist
		},
		Env: func(string) string { return "" },
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	})
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}
