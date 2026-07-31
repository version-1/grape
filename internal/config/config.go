package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-1/grape/internal/worktree"
)

type ReadFileFunc func(string) ([]byte, error)

type ResetConfig struct {
	DefaultBranch string                    `json:"default_branch"`
	Worktrees     []worktree.ConfiguredItem `json:"worktrees"`
}

var ErrConfigAlreadyExists = errors.New("config already exists")

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

	currentPath := "grape.json"
	if _, err := r.Stat(currentPath); err == nil {
		return currentPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	grapeHome, err := r.GrapeHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(grapeHome, "grape.json"), nil
}

func (r PathResolver) GrapeHome() (string, error) {
	if grapeHome := r.Env("GRAPE_HOME"); grapeHome != "" {
		return grapeHome, nil
	}

	home, err := r.UserHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".grape"), nil
}

// Initialize writes the example configuration to destination without replacing
// an existing configuration file.
func Initialize(examplePath string, destination string) error {
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("read example config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	return initializeConfigFile(destination, data, (*os.File).Write)
}

func initializeConfigFile(destination string, data []byte, write func(*os.File, []byte) (int, error)) error {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrConfigAlreadyExists
		}
		return fmt.Errorf("create config: %w", err)
	}

	initialized := false
	defer func() {
		if !initialized {
			_ = file.Close()
			_ = os.Remove(destination)
		}
	}()

	if _, err := write(file, data); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	initialized = true
	return nil
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
		path, err := filepath.Abs(item.Path)
		if err != nil {
			return ResetConfig{}, fmt.Errorf("resolve worktrees[%d].path: %w", i, err)
		}
		if _, ok := paths[path]; ok {
			return ResetConfig{}, fmt.Errorf("duplicate path %q", item.Path)
		}
		paths[path] = struct{}{}

		config.Worktrees[i].Branch = branch
	}

	return config, nil
}
