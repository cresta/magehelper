package githubactions

import (
	"context"
	"errors"
	"fmt"

	"github.com/cresta/magehelper/cicd"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/pipe"
	"github.com/sethvargo/go-githubactions"
)

type GithubActions struct {
	Env     *env.Env
	Actions *githubactions.Action
}

type Factory struct {
	Env *env.Env
}

func init() {
	var f Factory
	cicd.Register(f.New)
}

func (f *Factory) New() (cicd.CiCd, error) {
	if isGithubActions(f.Env) {
		return &GithubActions{
			Env:     f.Env,
			Actions: githubactions.New(),
		}, nil
	}
	return nil, nil
}

func isGithubActions(env *env.Env) bool {
	const trueStr = "true"
	return env.Get("CI") == trueStr && env.Get("GITHUB_ACTIONS") == trueStr
}

var _ cicd.CiCd = &GithubActions{}

func (g *GithubActions) IncrementalID() string {
	return g.Env.Get("GITHUB_RUN_NUMBER")
}

func (g *GithubActions) GitSHA() string {
	return g.Env.Get("GITHUB_SHA")
}

func (g *GithubActions) GitRef() string {
	return g.Env.Get("GITHUB_REF")
}

func (g *GithubActions) AddStepOutput(key string, value string) {
	g.Actions.SetOutput(key, value)
}

func (g *GithubActions) Name() string {
	return "gh"
}

func (g *GithubActions) GitRepository() string {
	return g.Env.Get("GITHUB_REPOSITORY")
}

// FreeDiskSpace will free up space on disk.  There is a lot of cruft on the github actions runners.  Here are a few
// interesting links:
//   - https://github.com/ThewBear/free-actions
//   - https://github.com/actions/virtual-environments/issues/709
//   - https://github.com/NickleDave/vak/issues/341
//   - https://github.com/flashlight/wav2letter/actions/runs/74797824/workflow
//   - https://github.com/search?q=%22rm+-rf+%2Fusr%2Fshare%2Fdotnet%22&type=code
func FreeDiskSpace(ctx context.Context) error {
	if _, isGh := cicd.Instance().(*GithubActions); !isGh {
		return fmt.Errorf("cicd is not set to github actions: %s", cicd.Instance().Name())
	}
	if !isGithubActions(nil) {
		return errors.New("do not appear to be running inside github actions")
	}
	if err := pipe.NewPiped("df", "-h").Run(ctx); err != nil {
		return err
	}
	if err := pipe.NewPiped("rm", "-rf", "/usr/share/dotnet", "/usr/local/lib/android", "/usr/share/swift").Run(ctx); err != nil {
		return err
	}
	return pipe.NewPiped("df", "-h").Run(ctx)
}
