package command

import (
	"fmt"
	"io"
)

type VersionCommand struct{ Version, Commit string }

func (c VersionCommand) Run(stdout io.Writer) int {
	fmt.Fprintf(stdout, "grape %s (%s)\n", c.Version, c.Commit)
	return 0
}
