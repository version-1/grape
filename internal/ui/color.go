package ui

import "github.com/version-1/grape/internal/color"

type Color = color.Code

const (
	Red    = color.Red
	Green  = color.Green
	Yellow = color.Yellow
	Blue   = color.Blue
	Cyan   = color.Cyan
	Bold   = color.Bold
	Dim    = color.Dim
)

func Paint(enabled bool, value string, colors ...Color) string {
	return color.Paint(enabled, value, colors...)
}
