package ecr

import (
	"context"
	"fmt"
	"os"

	"github.com/cresta/magehelper/docker/registry"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/pipe"
)

type Ecr struct {
	AwsDefaultRegion string
	AccountID        string
}

var _ registry.Registry = &Ecr{}

func New(e env.Env) *Ecr {
	return &Ecr{
		AwsDefaultRegion: e.GetDefault("AWS_DEFAULT_REGION", "us-west-2"),
		AccountID:        e.GetDefault("AWS_ACCOUNT_ID", "0"),
	}
}

func (e Ecr) ContainerRegistry() string {
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", e.AccountID, e.AwsDefaultRegion)
}

// Log into ECR
func (e Ecr) Login(ctx context.Context) error {
	p := pipe.NewPiped("aws", "ecr", "get-login-password", "--region", e.AwsDefaultRegion).Pipe("docker", "login", "--username=AWS", "--password-stdin", e.ContainerRegistry())
	return p.Execute(ctx, nil, os.Stdout, os.Stderr)
}
