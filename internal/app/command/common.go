package command

import (
	"errors"
	"io"
)

type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type ExitCodeError interface {
	ExitCode() int
}

func ExitCode(err error) (int, bool) {
	var exitErr ExitCodeError
	if !errors.As(err, &exitErr) {
		return 0, false
	}
	return exitErr.ExitCode(), true
}
