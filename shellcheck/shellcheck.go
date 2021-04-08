package shellcheck

import (
	"context"
	"fmt"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/pipe"
)

type ShellCheck struct{}

var instance ShellCheck

func (s *ShellCheck) Lint(ctx context.Context) error {
	// Find all *.sh files
	allSh, err := files.AllWithExtension(".sh")
	if err != nil {
		return err
	}
	if len(allSh) == 0 {
		fmt.Println("No shell scripts to lint")
		return nil
	}
	return pipe.NewPiped("docker", `run`, `--rm`, `-v`, `/h/goland/magehelper:/mnt:ro`, `koalaman/shellcheck:stable`, allSh[0]).Run(ctx)
}

// Lint all *.sh files with shellcheck
func Lint(ctx context.Context) error {
	return instance.Lint(ctx)
}
