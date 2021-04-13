package lambda

import (
	"context"

	"github.com/cresta/magehelper/files"

	"github.com/magefile/mage/mg"

	"github.com/cresta/magehelper/docker"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/pipe"
)

var Instance = &Lambda{}

type Lambda struct {
	Env    env.Env
	Docker *docker.Docker
}

func (l *Lambda) docker() *docker.Docker {
	if l == nil || l.Docker == nil {
		return docker.Instance
	}
	return l.Docker
}

func (l *Lambda) RunContainer(ctx context.Context) error {
	mg.Deps(docker.Build)
	args := []string{
		"run", "-p", "9000:8080",
	}
	envList := l.Env.GetDefault("DOCKER_RUN_ENV", "env.list")
	if files.FileExists(envList) {
		args = append(args, "--env-file", envList)
	}
	args = append(args, l.docker().Image(), "/main")
	return pipe.NewPiped("docker", args...).Run(ctx)
}

// Execute a docker container for this lambda, using lambda RIE
func RunContainer(ctx context.Context) error {
	return Instance.RunContainer(ctx)
}
