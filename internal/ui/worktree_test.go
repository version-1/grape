package ui

import (
	"strings"
	"testing"

	"github.com/version-1/grape/internal/worktree"
)

func TestFormatWorktreeList(t *testing.T) {
	got := FormatWorktreeList([]worktree.Worktree{
		{Path: "/repo", Head: "1111111111111111111111111111111111111111", Branch: "main"},
		{Path: "/repo-detached", Head: "2222222222222222222222222222222222222222"},
	}, false)

	assertContains(t, got, "PATH")
	assertContains(t, got, "BRANCH")
	assertContains(t, got, "/repo-detached")
	assertContains(t, got, "(detached)")
	assertContains(t, got, "111111111111")
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}
