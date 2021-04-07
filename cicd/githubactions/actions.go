package githubactions

import (
	"github.com/cresta/magehelper/cicd"
	"github.com/cresta/magehelper/env"
)

type GithubActions struct {
	Env *env.Env
}

type Factory struct {
	Env *env.Env
}

func init() {
	var f Factory
	cicd.Register(f.New)
}

func (f *Factory) New() (cicd.CiCd, error) {
	if f.Env.Get("GITHUB_ACTIONS") == "true" {
		return &GithubActions{
			Env: f.Env,
		}, nil
	}
	return nil, nil
}

var _ cicd.CiCd = &GithubActions{}

func (g *GithubActions) IncrementalId() string {
	return g.Env.Get("GITHUB_RUN_NUMBER")
}

func (g *GithubActions) GitSHA() string {
	return g.Env.Get("GITHUB_SHA")
}

func (g *GithubActions) GitRef() string {
	return g.Env.Get("GITHUB_REF")
}

func (g *GithubActions) Name() string {
	return "gh"
}

func (g *GithubActions) GitRepository() string {
	return g.Env.Get("GITHUB_REPOSITORY")
}
