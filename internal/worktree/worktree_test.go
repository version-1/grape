package worktree

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
)

type commandExitError int

func (e commandExitError) Error() string { return "exit" }
func (e commandExitError) ExitCode() int { return int(e) }

func TestParse(t *testing.T) {
	output := strings.Join([]string{
		"worktree /repo",
		"HEAD 1111111111111111111111111111111111111111",
		"branch refs/heads/main",
		"",
		"worktree /repo-feature",
		"HEAD 2222222222222222222222222222222222222222",
		"branch refs/heads/feature/example",
		"",
		"worktree /repo-detached",
		"HEAD 3333333333333333333333333333333333333333",
		"detached",
	}, "\n")

	got := Parse(output)
	want := []Worktree{
		{Path: "/repo", Head: "1111111111111111111111111111111111111111", Branch: "main", Main: true},
		{Path: "/repo-feature", Head: "2222222222222222222222222222222222222222", Branch: "feature/example"},
		{Path: "/repo-detached", Head: "3333333333333333333333333333333333333333"},
	}

	if !slices.Equal(got, want) {
		t.Fatalf("worktrees = %#v, want %#v", got, want)
	}
}

func TestNewMatcherMatchesPrefix(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "worktrees")
	matchingPath := filepath.Join(target, "repo-feature")
	otherPath := filepath.Join(root, "other", "repo-feature")

	matcher, err := NewMatcher(MatchOptions{Pattern: target})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	got := Filter([]Worktree{{Path: matchingPath}, {Path: otherPath}}, matcher)
	if !slices.Equal(got, []Worktree{{Path: matchingPath}}) {
		t.Fatalf("got = %#v, want only matching path", got)
	}
}

func TestNewMatcherMatchesRegex(t *testing.T) {
	matcher, err := NewMatcher(MatchOptions{Pattern: `feature-(one|two)$`, Regex: true})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	got := Filter([]Worktree{{Path: "/repo-feature-one"}, {Path: "/repo-main"}}, matcher)
	if !slices.Equal(got, []Worktree{{Path: "/repo-feature-one"}}) {
		t.Fatalf("got = %#v, want regex match", got)
	}
}

func TestCommandClientPushUsesExactArguments(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		want  []string
	}{
		{"normal", false, []string{"push", "origin", "HEAD:feature/test"}},
		{"force with lease", true, []string{"push", "--force-with-lease", "origin", "HEAD:feature/test"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			client := CommandClient{RunCommand: func(_ context.Context, args []string, _ io.Reader, _ io.Writer, _ io.Writer) error {
				got = append([]string(nil), args...)
				return nil
			}}
			if err := client.Push(context.Background(), "feature/test", test.force, io.Discard, io.Discard); err != nil {
				t.Fatalf("Push() error = %v", err)
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("args = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestCommandClientRemoveWorktreeUsesExactArguments(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		want  []string
	}{
		{"normal", false, []string{"worktree", "remove", "/repo-feature"}},
		{"force", true, []string{"worktree", "remove", "--force", "/repo-feature"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			client := CommandClient{RunCommand: func(_ context.Context, args []string, _ io.Reader, _ io.Writer, _ io.Writer) error {
				got = append([]string(nil), args...)
				return nil
			}}
			if err := client.RemoveWorktree(context.Background(), "/repo-feature", test.force, io.Discard, io.Discard); err != nil {
				t.Fatalf("RemoveWorktree() error = %v", err)
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("args = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestCommandClientOriginURLSelection(t *testing.T) {
	tests := []struct {
		name      string
		pushURLs  string
		urls      string
		want      string
		wantError string
	}{
		{"push URL precedence", "ssh://push\n", "https://fetch\n", "ssh://push", ""},
		{"URL fallback", "", "https://fetch\n", "https://fetch", ""},
		{"missing", "", "", "", "remote.origin.url is missing"},
		{"multiple push URLs", "one\ntwo\n", "fallback\n", "", "remote.origin.pushurl must have exactly one value"},
		{"multiple URLs", "", "one\ntwo\n", "", "remote.origin.url must have exactly one value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := CommandClient{RunCommand: configCommandRunner(test.pushURLs, test.urls)}
			got, err := client.OriginURL(context.Background())
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("OriginURL() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("URL = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCommandClientOriginURLCountsEmptyConfiguredValues(t *testing.T) {
	tests := []struct {
		name      string
		pushValue string
		urlValue  string
		key       string
		wantError string
	}{
		{"empty push URL", "\n", "fallback\n", "remote.origin.pushurl", "is empty"},
		{"empty plus push URL", "\nssh://push\n", "fallback\n", "remote.origin.pushurl", "must have exactly one value"},
		{"empty URL", "", "\n", "remote.origin.url", "is empty"},
		{"empty plus URL", "", "\nhttps://fetch\n", "remote.origin.url", "must have exactly one value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := CommandClient{RunCommand: func(_ context.Context, args []string, _ io.Reader, stdout io.Writer, _ io.Writer) error {
				key := args[len(args)-1]
				value := test.urlValue
				if key == "remote.origin.pushurl" {
					value = test.pushValue
				}
				if value == "" {
					return commandExitError(1)
				}
				_, _ = io.WriteString(stdout, value)
				return nil
			}}
			_, err := client.OriginURL(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.key) || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("OriginURL() error = %v, want key %q and %q", err, test.key, test.wantError)
			}
		})
	}
}

func configCommandRunner(pushURLs string, urls string) RunGitFunc {
	return func(_ context.Context, args []string, _ io.Reader, stdout io.Writer, _ io.Writer) error {
		var value string
		switch args[len(args)-1] {
		case "remote.origin.pushurl":
			value = pushURLs
		case "remote.origin.url":
			value = urls
		}
		if value == "" {
			return commandExitError(1)
		}
		_, _ = io.WriteString(stdout, value)
		return nil
	}
}

func TestCommandClientCurrentAndBranchValidation(t *testing.T) {
	var calls [][]string
	client := CommandClient{RunCommand: func(_ context.Context, args []string, _ io.Reader, stdout io.Writer, _ io.Writer) error {
		calls = append(calls, append([]string(nil), args...))
		if args[0] == "symbolic-ref" {
			_, _ = io.WriteString(stdout, "feature/test\n")
		}
		return nil
	}}

	branch, err := client.CurrentBranch(context.Background())
	if err != nil || branch != "feature/test" {
		t.Fatalf("CurrentBranch() = %q, %v", branch, err)
	}
	if err := client.ValidateBranch(context.Background(), branch); err != nil {
		t.Fatalf("ValidateBranch() error = %v", err)
	}
	want := [][]string{
		{"symbolic-ref", "--quiet", "--short", "HEAD"},
		{"check-ref-format", "--branch", "feature/test"},
	}
	if !slices.EqualFunc(calls, want, slices.Equal[[]string]) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestCommandClientRejectsBranchBeginningWithDashWithoutGit(t *testing.T) {
	called := false
	client := CommandClient{RunCommand: func(context.Context, []string, io.Reader, io.Writer, io.Writer) error {
		called = true
		return nil
	}}
	if err := client.ValidateBranch(context.Background(), "-danger"); err == nil {
		t.Fatal("ValidateBranch() error = nil")
	}
	if called {
		t.Fatal("git was called")
	}
}

func TestCommandClientPreservesPushStreamsAndError(t *testing.T) {
	wantErr := errors.New("push failed")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	client := CommandClient{RunCommand: func(_ context.Context, _ []string, _ io.Reader, gotStdout io.Writer, gotStderr io.Writer) error {
		_, _ = io.WriteString(gotStdout, "raw stdout")
		_, _ = io.WriteString(gotStderr, "raw stderr")
		return wantErr
	}}

	err := client.Push(context.Background(), "feature/test", false, stdout, stderr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Push() error = %v, want %v", err, wantErr)
	}
	if stdout.String() != "raw stdout" || stderr.String() != "raw stderr" {
		t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
	}
}

func TestCommandClientRebaseUsesExactArgumentsAndPreservesStreams(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
	}{
		{"tracking upstream", nil, []string{"rebase"}},
		{"all arguments", []string{"--onto", "main", "base", "feature"}, []string{"rebase", "--onto", "main", "base", "feature"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wantErr := errors.New("rebase failed")
			stdin := strings.NewReader("input")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			var gotArgs []string
			client := CommandClient{RunCommand: func(_ context.Context, args []string, gotStdin io.Reader, gotStdout io.Writer, gotStderr io.Writer) error {
				gotArgs = append([]string(nil), args...)
				if gotStdin != stdin || gotStdout != stdout || gotStderr != stderr {
					t.Fatal("rebase streams were not preserved")
				}
				_, _ = io.WriteString(gotStdout, "raw stdout")
				_, _ = io.WriteString(gotStderr, "raw stderr")
				return wantErr
			}}

			err := client.Rebase(context.Background(), test.args, stdin, stdout, stderr)

			if !errors.Is(err, wantErr) {
				t.Fatalf("Rebase() error = %v, want %v", err, wantErr)
			}
			if !slices.Equal(gotArgs, test.wantArgs) {
				t.Fatalf("args = %#v, want %#v", gotArgs, test.wantArgs)
			}
			if stdout.String() != "raw stdout" || stderr.String() != "raw stderr" {
				t.Fatalf("stdout = %q, stderr = %q", stdout, stderr)
			}
		})
	}
}

func TestCommandClientRebaseBranchReadsActiveRebaseState(t *testing.T) {
	tests := []struct {
		name      string
		available string
		want      string
	}{
		{"merge backend", "rebase-merge/head-name", "feature/merge"},
		{"apply backend", "rebase-apply/head-name", "feature/apply"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := CommandClient{
				RunCommand: func(_ context.Context, args []string, _ io.Reader, stdout io.Writer, _ io.Writer) error {
					_, _ = io.WriteString(stdout, args[2])
					return nil
				},
				ReadFile: func(path string) ([]byte, error) {
					if path != test.available {
						return nil, os.ErrNotExist
					}
					return []byte("refs/heads/" + test.want + "\n"), nil
				},
			}

			got, err := client.RebaseBranch(context.Background())
			if err != nil {
				t.Fatalf("RebaseBranch() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("branch = %q, want %q", got, test.want)
			}
		})
	}
}
