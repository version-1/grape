package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/version-1/grape/internal/worktree"
)

var (
	ErrConfigAlreadyExists = errors.New("config already exists")
	ErrConfigNotFound      = errors.New("config not found")
)

var defaultProtectedBranches = []string{"main", "master"}

type ReadFileFunc func(string) ([]byte, error)

type Config struct {
	DefaultBranch string                    `json:"default_branch,omitempty"`
	Worktrees     []worktree.ConfiguredItem `json:"worktrees,omitempty"`
	Push          *PushConfig               `json:"push,omitempty"`
}

type PushConfig struct {
	ProtectedBranches *[]string `json:"protected_branches,omitempty"`
}

type Discovery struct {
	Path  string
	Found bool
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

func (r PathResolver) Discover(explicitPath string) (Discovery, error) {
	if explicitPath != "" {
		return Discovery{Path: explicitPath, Found: true}, nil
	}

	currentPath := "grape.json"
	if _, err := r.Stat(currentPath); err == nil {
		return Discovery{Path: currentPath, Found: true}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Discovery{}, err
	}

	grapeHome, err := r.GrapeHome()
	if err != nil {
		return Discovery{}, err
	}
	homePath := filepath.Join(grapeHome, "grape.json")
	if _, err := r.Stat(homePath); err == nil {
		return Discovery{Path: homePath, Found: true}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Discovery{}, err
	}
	return Discovery{Path: homePath}, nil
}

func (r PathResolver) ResolveConfigPath(explicitPath string) (string, error) {
	result, err := r.Discover(explicitPath)
	if err != nil {
		return "", err
	}
	return result.Path, nil
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

func Read(path string, readFile ReadFileFunc) (Config, error) {
	data, err := readFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := validateJSONDocument(data); err != nil {
		return Config{}, err
	}

	var config Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, err
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Config{}, err
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func validateJSONDocument(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := validateJSONValue(decoder, "$"); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("multiple JSON values")
	}
	return nil
}

func validateJSONValue(decoder *json.Decoder, location string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("%s must not be null", location)
	}

	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%s has a non-string object key", location)
			}
			if !allowedJSONField(location, key) {
				return fmt.Errorf("%s has unknown field %q", location, key)
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("%s has duplicate field %q", location, key)
			}
			keys[key] = struct{}{}
			if err := validateJSONValue(decoder, location+"."+key); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			if err := validateJSONValue(decoder, fmt.Sprintf("%s[%d]", location, index)); err != nil {
				return err
			}
			index++
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("%s has unexpected delimiter %q", location, delimiter)
	}
}

func allowedJSONField(location string, key string) bool {
	switch {
	case location == "$":
		return key == "default_branch" || key == "worktrees" || key == "push"
	case location == "$.push":
		return key == "protected_branches"
	case strings.HasPrefix(location, "$.worktrees[") && strings.HasSuffix(location, "]"):
		return key == "path" || key == "branch" || key == "start_point"
	default:
		return false
	}
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("multiple JSON values")
}

func (c Config) Validate() error {
	for i, pattern := range c.ProtectedBranches() {
		if pattern == "" {
			return fmt.Errorf("push.protected_branches[%d] is empty", i)
		}
		if _, err := path.Match(pattern, "branch"); err != nil {
			return fmt.Errorf("push.protected_branches[%d]: %w", i, err)
		}
	}
	if c.Push != nil && c.Push.ProtectedBranches != nil && len(*c.Push.ProtectedBranches) == 0 {
		return errors.New("push.protected_branches must not be empty")
	}
	return nil
}

func (c *Config) ValidateReset() error {
	if len(c.Worktrees) == 0 {
		return errors.New("worktrees must not be empty")
	}

	branches := make(map[string]struct{}, len(c.Worktrees))
	paths := make(map[string]struct{}, len(c.Worktrees))
	for i, item := range c.Worktrees {
		if item.Path == "" {
			return fmt.Errorf("worktrees[%d].path is empty", i)
		}
		if item.Branch == "" {
			return fmt.Errorf("worktrees[%d].branch is empty", i)
		}
		branch := worktree.NormalizeBranchName(item.Branch)
		if _, ok := branches[branch]; ok {
			return fmt.Errorf("duplicate branch %q", branch)
		}
		branches[branch] = struct{}{}
		absolutePath, err := filepath.Abs(item.Path)
		if err != nil {
			return fmt.Errorf("resolve worktrees[%d].path: %w", i, err)
		}
		if _, ok := paths[absolutePath]; ok {
			return fmt.Errorf("duplicate path %q", item.Path)
		}
		paths[absolutePath] = struct{}{}
		c.Worktrees[i].Branch = branch
	}
	return nil
}

func (c Config) ProtectedBranches() []string {
	if c.Push == nil || c.Push.ProtectedBranches == nil {
		return append([]string(nil), defaultProtectedBranches...)
	}
	return append([]string(nil), (*c.Push.ProtectedBranches)...)
}

func (c Config) BranchProtected(branch string) bool {
	for _, pattern := range c.ProtectedBranches() {
		matched, _ := path.Match(pattern, branch)
		if matched {
			return true
		}
	}
	return false
}

func ReadResetConfig(path string, readFile ReadFileFunc) (Config, error) {
	config, err := Read(path, readFile)
	if err != nil {
		return Config{}, err
	}
	if err := config.ValidateReset(); err != nil {
		return Config{}, err
	}
	return config, nil
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
