package logging

import (
	"fmt"
	"io"

	"github.com/version-1/grape/internal/color"
)

type Logger struct {
	writer io.Writer
	color  bool
}

func New(writer io.Writer, colorEnabled bool) Logger {
	return Logger{writer: writer, color: colorEnabled}
}

func (l Logger) Info(format string, args ...any) {
	fmt.Fprintf(l.writer, "grape: "+format+"\n", args...)
}

func (l Logger) Warning(format string, args ...any) {
	prefix := color.Paint(l.color, "grape: warning:", color.Yellow)
	fmt.Fprintf(l.writer, prefix+" "+format+"\n", args...)
}

func (l Logger) Error(format string, args ...any) {
	prefix := color.Paint(l.color, "grape:", color.Red)
	fmt.Fprintf(l.writer, prefix+" "+format+"\n", args...)
}
