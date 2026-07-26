package ui

import (
	"fmt"
	"strings"

	"github.com/version-1/grape/internal/worktree"
)

func FormatWorktreeList(worktrees []worktree.Worktree) string {
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
		Paint(padRight("PATH", pathWidth), Bold),
		Paint(padRight("BRANCH", branchWidth), Bold),
		Paint("HEAD", Bold),
	)
	for _, item := range worktrees {
		fmt.Fprintf(
			&builder,
			"%s  %s  %s\n",
			Paint(padRight(item.Path, pathWidth), Cyan),
			Paint(padRight(displayBranch(item), branchWidth), Green),
			Paint(shortHead(item.Head), Dim),
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
