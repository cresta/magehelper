package gobuild

import (
	"context"
	"fmt"
	"os"

	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/pipe"
)

var Instance Go

type Go struct {
	Env         env.Env
	BuildBinary string
}

func (g *Go) Lint(ctx context.Context) error {
	return pipe.NewPiped("golangci-lint", "run").Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) Test(ctx context.Context) error {
	return pipe.NewPiped("go", "test", "-v", "-race", "-benchtime", "1ns", "-bench", ".", "./...").
		WithEnv(g.Env.AddEnv("GORACE=halt_on_error=1")).
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) IntegrationTest(ctx context.Context) error {
	return pipe.NewPiped("go", "test", "--tags=integration ", "-v", "-race", "-benchtime", "1ns", "-bench", ".", "./...").
		WithEnv(g.Env.AddEnv("GORACE=halt_on_error=1")).
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) Reformat(ctx context.Context) error {
	err := pipe.NewPiped("gofmt", "-s", "-w", "./..").Execute(ctx, nil, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("unable to gofmt: %w", err)
	}
	return pipe.NewPiped("find", ".", "-iname", "*.go", "-print0").
		Pipe("xargs", "-0", "goimports", "-w").
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (g *Go) Build(ctx context.Context) error {
	if g.BuildBinary == "" {
		return fmt.Errorf("unset build target: change mage file")
	}
	return pipe.NewPiped("go", "build", "-o", "main", "-ldflags", `'-extldflags "-f no-PIC -static"'`, "-tags", "'osusergo netgo static_build'", g.BuildBinary).
		WithEnv(g.Env.AddEnv("GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")).
		Execute(ctx, nil, os.Stdout, os.Stderr)
}

// Build the current go code
func Build(ctx context.Context) error {
	return Instance.Build(ctx)
}

// Reformat the current go code
func Reformat(ctx context.Context) error {
	return Instance.Reformat(ctx)
}

// Lint the current go code
func Lint(ctx context.Context) error {
	return Instance.Lint(ctx)
}

// Test the current go code
func Test(ctx context.Context) error {
	return Instance.Test(ctx)
}

// IntegrationTest run against the current library
func IntegrationTest(ctx context.Context) error {
	return Instance.IntegrationTest(ctx)
}
