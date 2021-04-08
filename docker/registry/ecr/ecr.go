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
	Env              env.Env
	AwsDefaultRegion string
	AccountID        string
}

var Instance Ecr

var _ registry.Registry = &Ecr{}

func (e *Ecr) defaultRegion() string {
	if e.AwsDefaultRegion != "" {
		return e.AwsDefaultRegion
	}
	return e.Env.GetDefault("AWS_DEFAULT_REGION", "us-west-2")
}

func (e *Ecr) accountID() string {
	if e.AccountID != "" {
		return e.AccountID
	}
	return e.Env.GetDefault("AWS_ACCOUNT_ID", "0")
}

func (e *Ecr) ContainerRegistry() string {
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", e.accountID(), e.defaultRegion())
}

func (e *Ecr) Login(ctx context.Context) error {
	p := pipe.NewPiped("aws", "ecr", "get-login-password", "--region", e.defaultRegion()).Pipe("docker", "login", "--username=AWS", "--password-stdin", e.ContainerRegistry())
	return p.Execute(ctx, nil, os.Stdout, os.Stderr)
}

// Login will log into ECR using the AWS CLI
func Login(ctx context.Context) error {
	return Instance.Login(ctx)
}
