package ui

import "strings"

type Color string

const (
	Reset Color = "\033[0m"
	Red   Color = "\033[31m"
	Green Color = "\033[32m"
	Blue  Color = "\033[34m"
	Cyan  Color = "\033[36m"
	Bold  Color = "\033[1m"
	Dim   Color = "\033[2m"
)

func Paint(value string, colors ...Color) string {
	var builder strings.Builder
	for _, color := range colors {
		builder.WriteString(string(color))
	}
	builder.WriteString(value)
	builder.WriteString(string(Reset))
	return builder.String()
}
