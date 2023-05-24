package yq

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/pipe"
)

type Yq struct{}

var Instance Yq

func (y *Yq) Reformat(ctx context.Context, path string) error {
	err := pipe.NewPiped("yq", "-P", "-i", path).Run(ctx)
	if err != nil {
		return fmt.Errorf("unable to reformat %s: %w", path, err)
	}
	return nil
}

func (y *Yq) VersionCheck(ctx context.Context) error {
	var out bytes.Buffer
	err := pipe.NewPiped("yq", "--version").Execute(ctx, nil, &out, nil)
	if err != nil {
		return fmt.Errorf("unable to check version: %w", err)
	}
	version := strings.TrimSpace(out.String())
	rgx := `.*mikefarah.* version v?4\..*`
	cmp := regexp.MustCompile(rgx)
	if !cmp.MatchString(version) {
		return fmt.Errorf("must match version 4 (%s), saw -- %s", rgx, version)
	}
	return nil
}

func (y *Yq) ReformatYAMLDir(ctx context.Context, root string) error {
	if err := y.VersionCheck(ctx); err != nil {
		return fmt.Errorf("the YQ commands require yq verions 4: %w", err)
	}
	yamlFiles, err := files.AllWithExtensionInDir(root, ".yaml")
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %w", root, err)
	}
	for _, file := range yamlFiles {
		err := y.Reformat(ctx, filepath.Join(root, file))
		if err != nil {
			return fmt.Errorf("unable to reformat %s: %w", file, err)
		}
	}
	return nil
}

func (y *Yq) TrimTrailingWhitespace(ctx context.Context, path string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %w", path, err)
	}

	lines := strings.Split(string(input), "\n")
	var newLines []string
	for _, line := range lines {
		newLines = append(newLines, strings.TrimRight(line, " "))
	}

	output := []byte(strings.Join(newLines, "\n"))

	if !bytes.Equal(input, output) {
		err = os.WriteFile(path, output, 0644)
		if err != nil {
			return fmt.Errorf("unable to write file %s: %w", path, err)
		}
	}

	return nil
}

func (y *Yq) TrimTrailingWhitespaceForYAMLDir(ctx context.Context, root string) error {
	yamlFiles, err := files.AllWithExtensionInDir(root, ".yaml")
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %w", root, err)
	}

	for _, file := range yamlFiles {
		err := y.TrimTrailingWhitespace(ctx, filepath.Join(root, file))
		if err != nil {
			return fmt.Errorf("unable to trim trailing whitespace %s: %w", file, err)
		}
	}

	return nil
}

// Reformat all YAML files in PATH with YQ
func Reformat(ctx context.Context, path string) error {
	return Instance.Reformat(ctx, path)
}

// VersionCheck checks the version of YQ to match 4.x
func VersionCheck(ctx context.Context) error {
	return Instance.VersionCheck(ctx)
}

// ReformatYAMLDir reformats all YAML files in PATH with YQ
func ReformatYAMLDir(ctx context.Context, root string) error {
	return Instance.ReformatYAMLDir(ctx, root)
}

func TrimTrailingWhitespace(ctx context.Context, path string) error {
	return Instance.TrimTrailingWhitespace(ctx, path)
}

// TrimTrailingWhitespaceForYAMLDir trims trailing whitespace for all YAML files in PATH
func TrimTrailingWhitespaceForYAMLDir(ctx context.Context, root string) error {
	return Instance.TrimTrailingWhitespaceForYAMLDir(ctx, root)
}
