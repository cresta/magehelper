package docker

import (
	"testing"

	"github.com/cresta/magehelper/cicd/githubactions"
	"github.com/cresta/magehelper/env"
	"github.com/stretchr/testify/require"
)

func TestDocker_Tag(t *testing.T) {
	e := env.NewFromMap(map[string]string{
		"GITHUB_REF":        "hotfix/fix-wrong-sha",
		"GITHUB_RUN_NUMBER": "123",
		"GITHUB_SHA":        "deadbeaf",
	})
	g := &githubactions.GithubActions{
		Env: e,
	}
	d := Docker{
		Env:  *e,
		CiCd: g,
	}
	require.Equal(t, "hotfix_fix-wrong-sha-gh.123-deadbea", d.Tag())
}
