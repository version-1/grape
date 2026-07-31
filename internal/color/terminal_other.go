//go:build !darwin && !linux

package color

import "os"

func fileIsTerminal(*os.File) bool {
	return false
}
