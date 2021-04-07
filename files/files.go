package files

import (
	"os"
	"path/filepath"
)

// IsDir returns true if the given path is an existing directory.
func IsDir(pathFile string) bool {
	if pathAbs, err := filepath.Abs(pathFile); err != nil {
		return false
	} else if fileInfo, err := os.Stat(pathAbs); os.IsNotExist(err) || !fileInfo.IsDir() {
		return false
	}

	return true
}
