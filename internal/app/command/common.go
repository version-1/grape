package command

import "io"

type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type ExitCodeError interface {
	ExitCode() int
}

func ExitCode(err error) (int, bool) {
	exitErr, ok := err.(ExitCodeError)
	if !ok {
		return 0, false
	}
	return exitErr.ExitCode(), true
}
