package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Matcher func(Worktree) bool

type MatchOptions struct {
	Pattern string
	Regex   bool
}

func NewMatcher(options MatchOptions) (Matcher, error) {
	if options.Regex {
		re, err := regexp.Compile(options.Pattern)
		if err != nil {
			return nil, err
		}
		return func(item Worktree) bool {
			return re.MatchString(item.Path)
		}, nil
	}

	prefix, err := normalizedPathPrefix(options.Pattern)
	if err != nil {
		return nil, err
	}
	return func(item Worktree) bool {
		path, err := normalizedPathPrefix(item.Path)
		if err != nil {
			return false
		}
		return path == prefix || strings.HasPrefix(path, prefix+string(os.PathSeparator))
	}, nil
}

func Filter(worktrees []Worktree, matcher Matcher) []Worktree {
	targets := make([]Worktree, 0)
	for _, item := range worktrees {
		if matcher(item) {
			targets = append(targets, item)
		}
	}
	return targets
}

func normalizedPathPrefix(path string) (string, error) {
	if path == "" {
		return "", errors.New("path prefix is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
