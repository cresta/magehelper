package docker

import (
	"fmt"
	"github.com/cresta/magehelper/cicd"
	"github.com/cresta/magehelper/docker/registry"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/git"
	"github.com/magefile/mage/sh"
	"regexp"
	"strings"
)

func trimLen(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[0:maxLen]
}

type Docker struct {
	Env env.Env
}

func (d Docker) Image(r registry.Registry, ci cicd.CiCd, g git.Git) string {
	return fmt.Sprintf("%s/%s:%s", r.ContainerRegistry(), d.Repository(ci, g), d.Tag(ci, g))
}

func (d Docker) SanitizeTag(s string) string {
	dockerRegex := regexp.MustCompile(`[^A-Za-z0-9_\.\-]`)
	// A tag name must be valid ASCII and may contain lowercase and uppercase letters, digits, underscores, periods and dashes. A tag name may not start with a period or a dash and may contain a maximum of 128 characters.
	// https://docs.docker.com/engine/reference/commandline/tag/#extended-description
	s = dockerRegex.ReplaceAllString(s, "_")
	if len(s) == 0 {
		return "latest"
	}
	if s[0] == '.' || s[0] == '-' {
		s = "_" + s
	}
	if len(s) > 128 {
		s = s[0:128]
	}
	return s
}

func (d Docker) Repository(ci cicd.CiCd, g git.Git) string {
	if d.Env.Get("DOCKER_REPOSITORY") != "" {
		return d.Env.Get("DOCKER_REPOSITORY")
	}
	if r := ci.GitRepository(); r != "" {
		return r
	}
	if r := g.RemoteRepository(); r != "" {
		return r
	}
	return "unknown/unknown"
}

func (d Docker) Tag(ci cicd.CiCd, g git.Git) string {
	ref := ci.GitRef()
	if ref == "" {
		ref = g.GitRef()
	}
	if strings.HasPrefix(ref, "refs/tags/") {
		t := strings.TrimPrefix(ref, "refs/tags/")
		if len(t) > 0 && t[0] == 'v' {
			t = t[1:]
		}
		return d.SanitizeTag(t)
	}
	branch := trimLen(g.BranchName(ref), 60)
	id := fmt.Sprintf("%s.%s", ci.Name(), ci.IncrementalId())
	sha := ci.GitSHA()
	if sha == "" {
		sha = g.GitSHA()
	}
	sha = trimLen(sha, 7)
	return d.SanitizeTag(fmt.Sprintf("%s-%s-%s", branch, id, sha))
}

func (d Docker) BuildxCacheFrom() string {
	return d.Env.GetDefault("DOCKER_BUILDX_FROM", "/tmp/.buildx-cache")
}

func (d Docker) BuildxCacheTo() string {
	return d.Env.GetDefault("DOCKER_BUILDX_TO", "/tmp/.buildx-cache-new")
}

// Build a docker image using buildx
func (d Docker) Build(r registry.Registry, ci cicd.CiCd, g git.Git) error {
	image := d.Image(r, ci, g)
	args := []string{"buildx", "build"}
	if d.Env.Get("DOCKER_PUSH") == "true" {
		args = append(args, "--push")
	} else {
		args = append(args, "--load")
	}
	cacheFrom := d.BuildxCacheFrom()
	cacheTo := d.BuildxCacheTo()
	if files.IsDir(cacheFrom) {
		args = append(args, fmt.Sprintf("--cache-from=type=local,src=%s", cacheFrom))
	}
	if files.IsDir(cacheTo) {
		args = append(args, fmt.Sprintf("--cache-from=type=local,src=%s", cacheTo))
	}
	args = append(args, fmt.Sprintf("--cache-to=type=local,dest=%s", cacheTo), "-t", image, ".")
	return sh.RunV("docker", args...)
}
