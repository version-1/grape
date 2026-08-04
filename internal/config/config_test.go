package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/version-1/grape/internal/worktree"
)

func TestReadResetConfig(t *testing.T) {
	config, err := ReadResetConfig("grape.json", func(string) ([]byte, error) {
		return []byte(`{
			"default_branch": "main",
			"worktrees": [
				{"path": "../repo-feature", "branch": "refs/heads/feature/example", "start_point": "origin/main"}
			]
		}`), nil
	})

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if config.DefaultBranch != "main" {
		t.Fatalf("DefaultBranch = %q, want main", config.DefaultBranch)
	}
	want := worktree.ConfiguredItem{Path: "../repo-feature", Branch: "feature/example", StartPoint: "origin/main"}
	if config.Worktrees[0] != want {
		t.Fatalf("worktree = %#v, want %#v", config.Worktrees[0], want)
	}
}

func TestReadResetConfigRejectsEmptyWorktrees(t *testing.T) {
	_, err := ReadResetConfig("grape.json", func(string) ([]byte, error) {
		return []byte(`{"worktrees":[]}`), nil
	})

	if err == nil {
		t.Fatal("err = nil, want error")
	}
	assertContains(t, err.Error(), "worktrees must not be empty")
}

func TestReadResetConfigRejectsEquivalentPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	_, err = ReadResetConfig("grape.json", func(string) ([]byte, error) {
		return []byte(fmt.Sprintf(`{"worktrees":[{"path":"worktree","branch":"feature/one"},{"path":%q,"branch":"feature/two"}]}`, filepath.Join(cwd, "worktree"))), nil
	})

	if err == nil {
		t.Fatal("err = nil, want error")
	}
	assertContains(t, err.Error(), "duplicate path")
}

func TestReadRejectsUnknownFields(t *testing.T) {
	tests := []string{
		`{"unknown":true}`,
		`{"push":{"unknown":true}}`,
		`{"worktrees":[{"path":"../repo","branch":"feature/test","unknown":true}]}`,
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Read("grape.json", func(string) ([]byte, error) {
				return []byte(input), nil
			})
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("err = %v, want unknown field error", err)
			}
		})
	}
}

func TestReadRejectsNullAndDuplicateFields(t *testing.T) {
	tests := []string{
		`null`,
		`{"push":null}`,
		`{"push":{"protected_branches":null}}`,
		`{"push":{"protected_branches":["release/*"]},"push":{}}`,
		`{"push":{"protected_branches":["main"],"protected_branches":["release/*"]}}`,
		`{"push":{"protected_branches":["release/*"]},"Push":{}}`,
		`{"push":{"protected_branches":["main"],"Protected_Branches":["release/*"]}}`,
		`{"worktrees":[{"path":"one","Path":"two","branch":"feature/test"}]}`,
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := Read("grape.json", func(string) ([]byte, error) {
				return []byte(input), nil
			}); err == nil {
				t.Fatalf("Read(%s) error = nil, want error", input)
			}
		})
	}
}

func TestReadSupportsPushOnlyResetOnlyAndCombinedConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"push only", `{"push":{"protected_branches":["trunk"]}}`},
		{"reset only", `{"worktrees":[{"path":"../repo","branch":"feature/test"}]}`},
		{"combined", `{"worktrees":[{"path":"../repo","branch":"feature/test"}],"push":{"protected_branches":["trunk"]}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Read("grape.json", func(string) ([]byte, error) {
				return []byte(test.input), nil
			}); err != nil {
				t.Fatalf("Read() error = %v", err)
			}
		})
	}
}

func TestProtectedBranchesDefaultsAndReplacement(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want []string
	}{
		{"omitted push", Config{}, []string{"main", "master"}},
		{"omitted list", Config{Push: &PushConfig{}}, []string{"main", "master"}},
		{"configured replacement", Config{Push: &PushConfig{ProtectedBranches: slicePointer([]string{"trunk", "release/*"})}}, []string{"trunk", "release/*"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.cfg.ProtectedBranches(); !slices.Equal(got, test.want) {
				t.Fatalf("ProtectedBranches() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestReadRejectsEmptyAndInvalidProtectedBranches(t *testing.T) {
	tests := []string{
		`{"push":{"protected_branches":[]}}`,
		`{"push":{"protected_branches":["["]}}`,
	}
	for _, input := range tests {
		if _, err := Read("grape.json", func(string) ([]byte, error) {
			return []byte(input), nil
		}); err == nil {
			t.Fatalf("Read(%s) error = nil, want error", input)
		}
	}
}

func TestBranchProtectedUsesPathMatchSemantics(t *testing.T) {
	cfg := Config{Push: &PushConfig{ProtectedBranches: slicePointer([]string{"main", "release/*"})}}
	tests := []struct {
		branch string
		want   bool
	}{
		{"main", true},
		{"Main", false},
		{"release/1.0", true},
		{"release/series/1.0", false},
		{"feature/main", false},
	}
	for _, test := range tests {
		if got := cfg.BranchProtected(test.branch); got != test.want {
			t.Fatalf("BranchProtected(%q) = %t, want %t", test.branch, got, test.want)
		}
	}
}

func TestValidateResetIsCommandSpecific(t *testing.T) {
	pushOnly := Config{Push: &PushConfig{ProtectedBranches: slicePointer([]string{"trunk"})}}
	if err := pushOnly.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := pushOnly.ValidateReset(); err == nil {
		t.Fatal("ValidateReset() error = nil, want missing worktrees error")
	}
}

func slicePointer(values []string) *[]string {
	return &values
}

func TestResolveConfigPathUsesExplicitPath(t *testing.T) {
	resolver := PathResolver{}

	got, err := resolver.ResolveConfigPath("custom.json")

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "custom.json" {
		t.Fatalf("path = %q, want custom.json", got)
	}
}

func TestResolveConfigPathPrefersCurrentDirectory(t *testing.T) {
	resolver := PathResolver{
		Stat: func(path string) (os.FileInfo, error) {
			if path != "grape.json" {
				t.Fatalf("stat path = %q, want grape.json", path)
			}
			return nil, nil
		},
		Env: func(string) string {
			return "/env/grape"
		},
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	}

	got, err := resolver.ResolveConfigPath("")

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != "grape.json" {
		t.Fatalf("path = %q, want grape.json", got)
	}
}

func TestResolveConfigPathFallsBackToGrapeHome(t *testing.T) {
	resolver := PathResolver{
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		Env: func(key string) string {
			if key == "GRAPE_HOME" {
				return "/env/grape"
			}
			return ""
		},
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	}

	got, err := resolver.ResolveConfigPath("")

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != filepath.Join("/env/grape", "grape.json") {
		t.Fatalf("path = %q, want GRAPE_HOME config", got)
	}
}

func TestResolveConfigPathFallsBackToDefaultGrapeHome(t *testing.T) {
	resolver := PathResolver{
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		Env: func(string) string {
			return ""
		},
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	}

	got, err := resolver.ResolveConfigPath("")

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != filepath.Join("/home/user", ".grape", "grape.json") {
		t.Fatalf("path = %q, want default GRAPE_HOME config", got)
	}
}

func TestResolveConfigPathIgnoresLegacyGWHome(t *testing.T) {
	resolver := PathResolver{
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		Env: func(key string) string {
			if key == "GW_HOME" {
				return "/legacy/gw"
			}
			return ""
		},
		UserHome: func() (string, error) {
			return "/home/user", nil
		},
	}

	got, err := resolver.ResolveConfigPath("")

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != filepath.Join("/home/user", ".grape", "grape.json") {
		t.Fatalf("path = %q, want default grape home config", got)
	}
}

func TestResolveConfigPathReturnsStatError(t *testing.T) {
	statErr := errors.New("stat failed")
	resolver := PathResolver{
		Stat: func(string) (os.FileInfo, error) {
			return nil, statErr
		},
	}

	_, err := resolver.ResolveConfigPath("")

	if !errors.Is(err, statErr) {
		t.Fatalf("err = %v, want %v", err, statErr)
	}
}

func TestDiscoverDistinguishesMissingFromSelectedFile(t *testing.T) {
	tests := []struct {
		name      string
		stat      func(string) (os.FileInfo, error)
		wantPath  string
		wantFound bool
		wantError error
	}{
		{
			name: "current selected",
			stat: func(path string) (os.FileInfo, error) {
				if path == "grape.json" {
					return nil, nil
				}
				return nil, os.ErrNotExist
			},
			wantPath: "grape.json", wantFound: true,
		},
		{
			name: "home selected",
			stat: func(path string) (os.FileInfo, error) {
				if path == filepath.Join("/home/user", ".grape", "grape.json") {
					return nil, nil
				}
				return nil, os.ErrNotExist
			},
			wantPath: filepath.Join("/home/user", ".grape", "grape.json"), wantFound: true,
		},
		{
			name: "all missing",
			stat: func(string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			wantPath: filepath.Join("/home/user", ".grape", "grape.json"), wantFound: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := PathResolver{
				Stat: test.stat,
				Env:  func(string) string { return "" },
				UserHome: func() (string, error) {
					return "/home/user", nil
				},
			}
			got, err := resolver.Discover("")
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Discover() error = %v, want %v", err, test.wantError)
			}
			if got.Path != test.wantPath || got.Found != test.wantFound {
				t.Fatalf("Discover() = %#v, want path=%q found=%t", got, test.wantPath, test.wantFound)
			}
		})
	}
}

func TestInitializeCreatesConfig(t *testing.T) {
	temporaryDir := t.TempDir()
	destination := filepath.Join(temporaryDir, ".grape", "grape.json")
	want := []byte(`{"worktrees":[]}`)

	if err := Initialize(want, destination); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("config = %q, want %q", got, want)
	}
}

func TestInitializeDoesNotOverwriteExistingConfig(t *testing.T) {
	temporaryDir := t.TempDir()
	destination := filepath.Join(temporaryDir, "grape.json")
	want := []byte(`{"worktrees":["existing"]}`)
	if err := os.WriteFile(destination, want, 0o600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	err := Initialize([]byte(`{"worktrees":["example"]}`), destination)
	if !errors.Is(err, ErrConfigAlreadyExists) {
		t.Fatalf("Initialize() error = %v, want ErrConfigAlreadyExists", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("config = %q, want existing value %q", got, want)
	}
}

func TestInitializeRemovesPartialConfigAfterWriteFailure(t *testing.T) {
	temporaryDir := t.TempDir()
	destination := filepath.Join(temporaryDir, "grape.json")
	writeErr := errors.New("write failed")

	err := initializeConfigFile(destination, []byte(`{"worktrees":[]}`), func(file *os.File, _ []byte) (int, error) {
		if _, err := file.Write([]byte("partial")); err != nil {
			t.Fatalf("write partial config: %v", err)
		}
		return 0, writeErr
	})

	if !errors.Is(err, writeErr) {
		t.Fatalf("initializeConfigFile() error = %v, want %v", err, writeErr)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", destination, err)
	}
}

func TestReadRebaseConfigPolicyMatching(t *testing.T) {
	config, err := ReadRebaseConfig("grape.json", func(string) ([]byte, error) {
		return []byte(`{"rebase":{"allowed_branches":["feature/*","worktrees/?"]}}`), nil
	})
	if err != nil {
		t.Fatalf("ReadRebaseConfig() error = %v", err)
	}

	tests := []struct {
		branch string
		want   bool
	}{
		{"feature/test", true},
		{"prefix-feature/test", false},
		{"feature/test-suffix", true},
		{"feature/team/test", false},
		{"worktrees/3", true},
		{"worktrees/33", false},
	}
	for _, test := range tests {
		t.Run(test.branch, func(t *testing.T) {
			if got := config.BranchAllowedForRebase(test.branch); got != test.want {
				t.Fatalf("BranchAllowedForRebase(%q) = %t, want %t", test.branch, got, test.want)
			}
		})
	}
}

func TestReadRebaseConfigMissingAndEmptyPolicyDenyAll(t *testing.T) {
	configs := []string{
		`{}`,
		`{"rebase":{}}`,
		`{"rebase":{"allowed_branches":[]}}`,
	}
	for _, contents := range configs {
		config, err := ReadRebaseConfig("grape.json", func(string) ([]byte, error) {
			return []byte(contents), nil
		})
		if err != nil {
			t.Fatalf("ReadRebaseConfig(%s) error = %v", contents, err)
		}
		if config.BranchAllowedForRebase("feature/test") {
			t.Fatalf("config %s allowed feature/test", contents)
		}
	}
}

func TestReadRebaseConfigValidatesEveryPattern(t *testing.T) {
	_, err := ReadRebaseConfig("grape.json", func(string) ([]byte, error) {
		return []byte(`{"rebase":{"allowed_branches":["feature/*","["]}}`), nil
	})
	if err == nil {
		t.Fatal("ReadRebaseConfig() error = nil")
	}
	assertContains(t, err.Error(), "rebase.allowed_branches[1]")
}

func TestReadRebaseConfigRejectsInvalidFieldTypes(t *testing.T) {
	tests := []string{
		`{"rebase":[]}`,
		`{"rebase":{"allowed_branches":"feature/*"}}`,
		`{"rebase":{"allowed_branches":[1]}}`,
	}
	for _, contents := range tests {
		t.Run(contents, func(t *testing.T) {
			if _, err := ReadRebaseConfig("grape.json", func(string) ([]byte, error) {
				return []byte(contents), nil
			}); err == nil {
				t.Fatal("ReadRebaseConfig() error = nil")
			}
		})
	}
}

func TestRebaseDoubleStarHasNoRecursiveMeaning(t *testing.T) {
	config, err := ReadRebaseConfig("grape.json", func(string) ([]byte, error) {
		return []byte(`{"rebase":{"allowed_branches":["feature/**"]}}`), nil
	})
	if err != nil {
		t.Fatalf("ReadRebaseConfig() error = %v", err)
	}
	if !config.BranchAllowedForRebase("feature/test") {
		t.Fatal("feature/** did not match feature/test")
	}
	if config.BranchAllowedForRebase("feature/team/test") {
		t.Fatal("feature/** recursively matched feature/team/test")
	}
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}
