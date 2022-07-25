package yq

import (
	"bytes"
	"context"
	"fmt"
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
	rgx := `mikefarah.* version 4\..`
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
