package grape

import (
	"bytes"
	_ "embed"
)

//go:embed grape.example.json
var exampleConfig []byte

// ExampleConfig returns a copy of the configuration bundled with grape.
func ExampleConfig() []byte {
	return bytes.Clone(exampleConfig)
}
