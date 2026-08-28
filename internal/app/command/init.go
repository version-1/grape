package command

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/version-1/grape/internal/config"
	"github.com/version-1/grape/internal/logging"
)

type InitCommand struct {
	Resolver          config.PathResolver
	ExampleConfigPath string
}

func (c InitCommand) Run(args []string, stdout io.Writer, logger logging.Logger) int {
	if len(args) != 0 {
		logger.Error("usage: grape init")
		return 2
	}
	home, err := c.Resolver.GrapeHome()
	if err != nil {
		logger.Error("resolve config directory: %v", err)
		return 1
	}
	destination := filepath.Join(home, "grape.json")
	if err := config.Initialize(c.ExampleConfigPath, destination); err != nil {
		if errors.Is(err, config.ErrConfigAlreadyExists) {
			logger.Error("config already exists: %s", destination)
		} else {
			logger.Error("initialize config: %v", err)
		}
		return 1
	}
	fmt.Fprintf(stdout, "Created config at %s\n", destination)
	return 0
}
