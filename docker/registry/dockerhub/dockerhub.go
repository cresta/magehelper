package dockerhub

import (
	"context"
	"os"
	"strings"

	"github.com/cresta/magehelper/docker/registry"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/pipe"
)

type DockerHub struct {
	Env env.Env
}

func (d *DockerHub) Password() string {
	return d.Env.GetDefault("DOCKERHUB_PASSWORD", "PLEASE-SET-PASSWORD")
}

func (d *DockerHub) Username() string {
	if d := d.Env.Get("DOCKERHUB_USERNAME"); d != "" {
		return d
	}
	return "PLEASE-SET-USERNAME"
}

func (d *DockerHub) ContainerRegistry() string {
	return "docker.io"
}

func (d *DockerHub) Login(ctx context.Context) error {
	return pipe.NewPiped("docker", "login", "--username", d.Username(), "--password-stdin", d.ContainerRegistry()).Execute(ctx, strings.NewReader(d.Password()), os.Stdout, os.Stderr)
}

var Instance = &DockerHub{}

var _ registry.Registry = Instance
