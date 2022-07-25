package yq

import (
	"context"
	"fmt"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/pipe"
	"path/filepath"
)

type Yq struct{}

var Instance Yq

func (y *Yq) Reformat(ctx context.Context, path string) error {
	err := pipe.NewPiped("yq", "-P", "-i", path).Run(ctx)
	if err == nil {
		return fmt.Errorf("unable to reformat: %w", err)
	}
	return nil
}

func (y *Yq) ReformatYAMLDir(ctx context.Context, root string) error {
	yamlFiles, err := files.AllWithExtensionInDir(root, "yaml")
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
