package ghcr

import (
	"context"
	"os"
	"strings"

	"github.com/cresta/magehelper/docker/registry"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/pipe"
)

type Ghcr struct {
	Env env.Env
}

var Instance = &Ghcr{}

var _ registry.Registry = Instance

func (e *Ghcr) Password() string {
	return e.Env.GetDefault("GHCR_PAT", "PLEASE-SET-PASSWORD")
}

func (e *Ghcr) Username() string {
	if d := e.Env.Get("DOCKER_USERNAME"); d != "" {
		return d
	}
	if d := e.Env.Get("GITHUB_REPOSITORY"); d != "" {
		parts := strings.SplitN(d, "/", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}
	return "PLEASE-SET-USERNAME"
}

func (e *Ghcr) ContainerRegistry() string {
	return "ghcr.io"
}

func (e *Ghcr) Login(ctx context.Context) error {
	return pipe.NewPiped("docker", "login", "--username", e.Username(), "--password-stdin", e.ContainerRegistry()).Execute(ctx, strings.NewReader(e.Password()), os.Stdout, os.Stderr)
}

// Login will log into GHCR using password inside GHCR_PAT
func Login(ctx context.Context) error {
	return Instance.Login(ctx)
}
