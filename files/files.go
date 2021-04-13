package files

import (
	"os"
	"path/filepath"
	"strings"
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

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// AllWithExtension returns all file names with the extension 'ext' from the current working directory.  They are returned
// relative to that directory
func AllWithExtension(ext string) ([]string, error) {
	ext = strings.ToLower(ext)
	pathS, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var files []string
	if err := filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if strings.ToLower(filepath.Ext(f.Name())) == ext || strings.ToLower(f.Name()) == ext {
				if rel, err := filepath.Rel(pathS, path); err == nil {
					files = append(files, rel)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}
