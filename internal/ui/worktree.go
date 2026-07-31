package ui

import (
	"fmt"
	"strings"

	"github.com/version-1/grape/internal/worktree"
)

func FormatWorktreeList(worktrees []worktree.Worktree, colorEnabled bool) string {
	pathWidth := len("PATH")
	branchWidth := len("BRANCH")
	for _, item := range worktrees {
		pathWidth = max(pathWidth, len(item.Path))
		branchWidth = max(branchWidth, len(displayBranch(item)))
	}

	var builder strings.Builder
	fmt.Fprintf(
		&builder,
		"%s  %s  %s\n",
		Paint(colorEnabled, padRight("PATH", pathWidth), Bold),
		Paint(colorEnabled, padRight("BRANCH", branchWidth), Bold),
		Paint(colorEnabled, "HEAD", Bold),
	)
	for _, item := range worktrees {
		fmt.Fprintf(
			&builder,
			"%s  %s  %s\n",
			Paint(colorEnabled, padRight(item.Path, pathWidth), Cyan),
			Paint(colorEnabled, padRight(displayBranch(item), branchWidth), Green),
			Paint(colorEnabled, shortHead(item.Head), Dim),
		)
	}
	return builder.String()
}

func displayBranch(item worktree.Worktree) string {
	if item.Branch == "" {
		return "(detached)"
	}
	return item.Branch
}

func shortHead(head string) string {
	if len(head) <= 12 {
		return head
	}
	return head[:12]
}

func padRight(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}
