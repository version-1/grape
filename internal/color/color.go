package color

import (
	"io"
	"os"
	"strings"
)

type Code string

const (
	Reset  Code = "\033[0m"
	Red    Code = "\033[31m"
	Green  Code = "\033[32m"
	Yellow Code = "\033[33m"
	Blue   Code = "\033[34m"
	Cyan   Code = "\033[36m"
	Bold   Code = "\033[1m"
	Dim    Code = "\033[2m"
)

type Policy struct {
	Stdout bool
	Stderr bool
}

func Detect(stdout io.Writer, stderr io.Writer, lookupEnv func(string) string) Policy {
	return detect(stdout, stderr, lookupEnv, isTerminal)
}

func detect(stdout io.Writer, stderr io.Writer, lookupEnv func(string) string, terminal func(io.Writer) bool) Policy {
	if lookupEnv != nil && lookupEnv("NO_COLOR") != "" {
		return Policy{}
	}
	return Policy{
		Stdout: terminal(stdout),
		Stderr: terminal(stderr),
	}
}

func Paint(enabled bool, value string, codes ...Code) string {
	if !enabled || len(codes) == 0 {
		return value
	}

	var builder strings.Builder
	for _, code := range codes {
		builder.WriteString(string(code))
	}
	builder.WriteString(value)
	builder.WriteString(string(Reset))
	return builder.String()
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return fileIsTerminal(file)
}
