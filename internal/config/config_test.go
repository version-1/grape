package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("got %q, want it to contain %q", got, want)
	}
}
