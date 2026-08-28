package command

import (
	"github.com/version-1/grape/internal/config"
	"github.com/version-1/grape/internal/logging"
)

// ConfigLoader owns command configuration discovery and decoding policy.
type ConfigLoader struct {
	PathResolver config.PathResolver
	ReadFile     config.ReadFileFunc
}

func (l ConfigLoader) Load(explicitPath string, logger logging.Logger) (config.Config, int) {
	discovery, err := l.PathResolver.Discover(explicitPath)
	if err != nil {
		logger.Error("resolve config: %v", err)
		return config.Config{}, 2
	}
	if !discovery.Found {
		logger.Warning("grape.json not found; using default protected branches: main, master")
		return config.Config{}, 0
	}
	cfg, err := config.Read(discovery.Path, l.ReadFile)
	if err != nil {
		logger.Error("read config: %v", err)
		return config.Config{}, 2
	}
	return cfg, 0
}

func (l ConfigLoader) LoadRebase(explicitPath string, logger logging.Logger) (config.Config, int) {
	discovery, err := l.PathResolver.Discover(explicitPath)
	if err != nil {
		logger.Error("resolve config: %v", err)
		return config.Config{}, 2
	}
	if !discovery.Found {
		logger.Error("rebase config not found: %s", discovery.Path)
		return config.Config{}, 2
	}
	cfg, err := config.ReadRebaseConfig(discovery.Path, l.ReadFile)
	if err != nil {
		logger.Error("read config: %v", err)
		return config.Config{}, 2
	}
	return cfg, 0
}
