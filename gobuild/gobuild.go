package gobuild

import (
	"context"
	"fmt"
	"os"

	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/pipe"
)

var Instance Go

type Go struct {
	Env env.Env
}

func (g *Go) buildMainDirectory() string {
	if dir := g.Env.Get("GOBUILD_MAIN_DIRECTORY"); dir != "" {
		return dir
	}
	// Try to guess it. Should be something like ./cmd/<X> if X exists and is a directory
	if !files.IsDir("./cmd") {
		return ""
	}
	entries, err := os.ReadDir("./cmd")
	if err != nil {
		return ""
	}
	finalReturn := ""
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		if ent.Name() == "." || ent.Name() == ".." {
			continue
		}
		if finalReturn != "" {
			return ""
		}
		finalReturn = "./cmd/" + ent.Name()
	}
	return finalReturn
}

func (g *Go) Lint(ctx context.Context) error {
	return pipe.NewPiped("golangci-lint", "run").Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) Test(ctx context.Context) error {
	args := []string{"test", "-v", "-race", "-benchtime", "1ns", "-bench", "."}
	if profileOut := g.Env.Get("GO_COVERAGE"); profileOut != "" {
		args = append(args, "-coverprofile", profileOut)
	}
	args = append(args, "./...")
	return pipe.NewPiped("go", args...).
		WithEnv(g.Env.AddEnv("GORACE=halt_on_error=1")).
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) IntegrationTest(ctx context.Context) error {
	return pipe.NewPiped("go", "test", "--tags=integration ", "-v", "-race", "-benchtime", "1ns", "-bench", ".", "./...").
		WithEnv(g.Env.AddEnv("GORACE=halt_on_error=1")).
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) Reformat(ctx context.Context) error {
	err := pipe.NewPiped("gofmt", "-s", "-w", ".").Execute(ctx, nil, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("unable to gofmt: %w", err)
	}
	return pipe.NewPiped("find", ".", "-iname", "*.go", "-print0").
		Pipe("xargs", "-0", "goimports", "-w").
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) Build(ctx context.Context) error {
	if g.buildMainDirectory() == "" {
		return fmt.Errorf("unset build target: change mage file")
	}
	return pipe.NewPiped("go", "build", "-o", "main", "-ldflags", `-extldflags "-f no-PIC -static"`, "-tags", "osusergo netgo static_build", g.buildMainDirectory()).
		WithEnv(g.Env.AddEnv("GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")).
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

// Will build a static binary of the go program in the directory ${GOBUILD_MAIN_DIRECTORY}
func Build(ctx context.Context) error {
	return Instance.Build(ctx)
}

// Format the code in place
func Reformat(ctx context.Context) error {
	return Instance.Reformat(ctx)
}

// Lints the current code using golangci-lint
func Lint(ctx context.Context) error {
	return Instance.Lint(ctx)
}

// Run a 'go test' against all code in this repository
func Test(ctx context.Context) error {
	return Instance.Test(ctx)
}

// Run a 'go test' with the integration flag for code in this repository
func IntegrationTest(ctx context.Context) error {
	return Instance.IntegrationTest(ctx)
}
