package worktree

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

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
