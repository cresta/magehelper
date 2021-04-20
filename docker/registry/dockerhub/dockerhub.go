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
	return d.Env.GetDefault("DOCKERHUB_USERNAME", d.Env.GetDefault("DOCKER_USERNAME", "PLEASE-SET-USERNAME"))
}

func (d *DockerHub) ContainerRegistry() string {
	return "docker.io"
}

func (d *DockerHub) Login(ctx context.Context) error {
	return pipe.NewPiped("docker", "login", "--username", d.Username(), "--password-stdin", d.ContainerRegistry()).Execute(ctx, strings.NewReader(d.Password()), os.Stdout, os.Stderr)
}

// Login will log into dockerhub using password inside DOCKERHUB_PASSWORD
func Login(ctx context.Context) error {
	return Instance.Login(ctx)
}

var Instance = &DockerHub{}

var _ registry.Registry = Instance
