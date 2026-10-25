package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartBranchSynchronizesWithBareOrigin(t *testing.T) {
	origin, seed, clone := startTestRepositories(t)
	writeStartTestFile(t, seed, "remote.txt", "new remote commit")
	runStartGit(t, seed, "add", "remote.txt")
	runStartGit(t, seed, "commit", "-m", "remote")
	runStartGit(t, seed, "push", "origin", "main")
	remoteHead := strings.TrimSpace(runStartGit(t, origin, "rev-parse", "main"))
	runStartGit(t, clone, "tag", "main")

	client := CommandClient{}
	withStartWorkingDirectory(t, clone, func() {
		if err := client.StartBranch(context.Background(), "main", "feat/from-latest", os.Stdout, os.Stderr); err != nil {
			t.Fatalf("StartBranch() error = %v", err)
		}
		if got := strings.TrimSpace(runStartGit(t, clone, "rev-parse", "HEAD")); got != remoteHead {
			t.Fatalf("new branch HEAD = %s, want remote main %s", got, remoteHead)
		}
		if got := strings.TrimSpace(runStartGit(t, clone, "branch", "--show-current")); got != "feat/from-latest" {
			t.Fatalf("current branch = %q", got)
		}
	})
}

func TestStartBranchFetchesMissingDefaultFromSingleBranchClone(t *testing.T) {
	origin, seed, _ := startTestRepositories(t)
	runStartGit(t, seed, "switch", "-c", "other")
	writeStartTestFile(t, seed, "other.txt", "other")
	runStartGit(t, seed, "add", "other.txt")
	runStartGit(t, seed, "commit", "-m", "other")
	runStartGit(t, seed, "push", "-u", "origin", "other")
	clone := filepath.Join(t.TempDir(), "single-branch")
	runStartGit(t, filepath.Dir(origin), "clone", "--single-branch", "--branch", "other", origin, clone)

	client := CommandClient{}
	withStartWorkingDirectory(t, clone, func() {
		if err := client.StartBranch(context.Background(), "main", "feat/from-main", os.Stdout, os.Stderr); err != nil {
			t.Fatalf("StartBranch() error = %v", err)
		}
		if got := strings.TrimSpace(runStartGit(t, clone, "branch", "--show-current")); got != "feat/from-main" {
			t.Fatalf("current branch = %q", got)
		}
	})
}

func TestStartBranchRejectsDefaultBranchAheadOfOrigin(t *testing.T) {
	_, _, clone := startTestRepositories(t)
	writeStartTestFile(t, clone, "local.txt", "local commit")
	runStartGit(t, clone, "add", "local.txt")
	runStartGit(t, clone, "commit", "-m", "local")

	client := CommandClient{}
	withStartWorkingDirectory(t, clone, func() {
		err := client.StartBranch(context.Background(), "main", "feat/must-not-exist", os.Stdout, os.Stderr)
		if err == nil || !strings.Contains(err.Error(), "contains commits") {
			t.Fatalf("StartBranch() error = %v, want local ahead rejection", err)
		}
		if output := runStartGit(t, clone, "branch", "--list", "feat/must-not-exist"); output != "" {
			t.Fatalf("feature branch was created: %q", output)
		}
	})
}

func startTestRepositories(t *testing.T) (origin, seed, clone string) {
	t.Helper()
	root := t.TempDir()
	origin = filepath.Join(root, "origin.git")
	seed = filepath.Join(root, "seed")
	clone = filepath.Join(root, "clone")
	runStartGit(t, root, "init", "--bare", origin)
	runStartGit(t, root, "clone", origin, seed)
	runStartGit(t, seed, "switch", "-c", "main")
	runStartGit(t, seed, "config", "user.name", "Test User")
	runStartGit(t, seed, "config", "user.email", "test@example.com")
	writeStartTestFile(t, seed, "README", "seed")
	runStartGit(t, seed, "add", "README")
	runStartGit(t, seed, "commit", "-m", "seed")
	runStartGit(t, seed, "push", "-u", "origin", "main")
	runStartGit(t, origin, "symbolic-ref", "HEAD", "refs/heads/main")
	runStartGit(t, root, "clone", origin, clone)
	runStartGit(t, clone, "config", "user.name", "Test User")
	runStartGit(t, clone, "config", "user.email", "test@example.com")
	return origin, seed, clone
}

func writeStartTestFile(t *testing.T, directory, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runStartGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func withStartWorkingDirectory(t *testing.T, directory string, run func()) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	run()
}
