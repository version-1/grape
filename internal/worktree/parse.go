package worktree

import "strings"

func Parse(output string) []Worktree {
	records := strings.Split(strings.TrimSpace(output), "\n\n")
	worktrees := make([]Worktree, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record) == "" {
			continue
		}

		item := Worktree{Main: len(worktrees) == 0}
		for _, line := range strings.Split(record, "\n") {
			key, value, ok := strings.Cut(line, " ")
			if !ok {
				continue
			}
			switch key {
			case "worktree":
				item.Path = value
			case "HEAD":
				item.Head = value
			case "branch":
				item.Branch = NormalizeBranchName(value)
			}
		}

		if item.Path != "" {
			worktrees = append(worktrees, item)
		}
	}
	return worktrees
}

func NormalizeBranchName(branch string) string {
	return strings.TrimPrefix(branch, "refs/heads/")
}
