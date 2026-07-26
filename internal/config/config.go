package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-1/dotfiles/shared/commands/gw/internal/worktree"
)

type ReadFileFunc func(string) ([]byte, error)

type ResetConfig struct {
	DefaultBranch string                    `json:"default_branch"`
	Worktrees     []worktree.ConfiguredItem `json:"worktrees"`
}

type PathResolver struct {
	Stat     func(string) (os.FileInfo, error)
	Env      func(string) string
	UserHome func() (string, error)
}

func DefaultPathResolver() PathResolver {
	return PathResolver{
		Stat:     os.Stat,
		Env:      os.Getenv,
		UserHome: os.UserHomeDir,
	}
}

func (r PathResolver) ResolveConfigPath(explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}

	currentPath := "gw.json"
	if _, err := r.Stat(currentPath); err == nil {
		return currentPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	gwHome, err := r.GWHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(gwHome, "gw.json"), nil
}

func (r PathResolver) GWHome() (string, error) {
	if gwHome := r.Env("GW_HOME"); gwHome != "" {
		return gwHome, nil
	}

	home, err := r.UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gw"), nil
}

func ReadResetConfig(path string, readFile ReadFileFunc) (ResetConfig, error) {
	data, err := readFile(path)
	if err != nil {
		return ResetConfig{}, err
	}

	var config ResetConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ResetConfig{}, err
	}
	if len(config.Worktrees) == 0 {
		return ResetConfig{}, errors.New("worktrees must not be empty")
	}

	branches := make(map[string]struct{}, len(config.Worktrees))
	paths := make(map[string]struct{}, len(config.Worktrees))
	for i, item := range config.Worktrees {
		if item.Path == "" {
			return ResetConfig{}, fmt.Errorf("worktrees[%d].path is empty", i)
		}
		if item.Branch == "" {
			return ResetConfig{}, fmt.Errorf("worktrees[%d].branch is empty", i)
		}

		branch := worktree.NormalizeBranchName(item.Branch)
		if _, ok := branches[branch]; ok {
			return ResetConfig{}, fmt.Errorf("duplicate branch %q", branch)
		}
		branches[branch] = struct{}{}
		path := filepath.Clean(item.Path)
		if _, ok := paths[path]; ok {
			return ResetConfig{}, fmt.Errorf("duplicate path %q", item.Path)
		}
		paths[path] = struct{}{}

		config.Worktrees[i].Branch = branch
		config.Worktrees[i].Path = path
	}

	return config, nil
}
