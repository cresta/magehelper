package files

import (
	"fmt"
	"io/fs"
	"io/ioutil"
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

// DirectoriesInDirectory returns all directories inside a root path (not subdirectories)
func DirectoriesInDirectory(path string) ([]string, error) {
	fi, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory: %w", err)
	}
	ret := make([]string, 0, len(fi))
	for _, f := range fi {
		if !f.IsDir() {
			continue
		}
		ret = append(ret, f.Name())
	}
	return ret, nil
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
			if hasExt(f, ext) {
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

func hasExt(f fs.FileInfo, ext string) bool {
	return strings.ToLower(filepath.Ext(f.Name())) == ext || strings.ToLower(f.Name()) == ext
}

func AllWithExtensionExactlyInDir(ext string, dir string) ([]string, error) {
	ext = strings.ToLower(ext)
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory: %w", err)
	}
	var ret []string
	for _, f := range fi {
		if f.IsDir() {
			continue
		}
		if hasExt(f, ext) {
			ret = append(ret, f.Name())
		}
	}
	return ret, nil
}
