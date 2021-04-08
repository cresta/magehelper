package docker

import (
	"context"
	"fmt"
	"github.com/cresta/magehelper/pipe"
	"os"
	"regexp"
	"strings"

	"github.com/cresta/magehelper/cicd"
	"github.com/cresta/magehelper/docker/registry"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/git"
)

func trimLen(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[0:maxLen]
}

var Instance Docker

type Docker struct {
	Env      env.Env
	Registry registry.Registry
	CiCd     cicd.CiCd
	Git      *git.Git
}

func (d *Docker) registry() registry.Registry {
	if d.Registry == nil {
		panic("Please set docker registry")
	}
	return d.Registry
}

func (d *Docker) cicd() cicd.CiCd {
	if d.CiCd == nil {
		return cicd.Instance
	}
	return d.CiCd
}

func (d *Docker) git() *git.Git {
	if d.Git == nil {
		return &git.Instance
	}
	return d.Git
}

func (d *Docker) Image() string {
	return fmt.Sprintf("%s/%s:%s", d.registry().ContainerRegistry(), d.Repository(), d.Tag())
}

func (d *Docker) SanitizeTag(s string) string {
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

func (d *Docker) Repository() string {
	if d.Env.Get("DOCKER_REPOSITORY") != "" {
		return d.Env.Get("DOCKER_REPOSITORY")
	}
	if r := d.cicd().GitRepository(); r != "" {
		return r
	}
	if r := d.git().RemoteRepository(); r != "" {
		return r
	}
	return "unknown/unknown"
}

func (d *Docker) Tag() string {
	ref := d.cicd().GitRef()
	if ref == "" {
		ref = d.git().GitRef()
	}
	if strings.HasPrefix(ref, "refs/tags/") {
		t := strings.TrimPrefix(ref, "refs/tags/")
		if len(t) > 0 && t[0] == 'v' {
			t = t[1:]
		}
		return d.SanitizeTag(t)
	}
	branch := trimLen(d.git().BranchName(ref), 60)
	id := fmt.Sprintf("%s.%s", d.cicd().Name(), d.cicd().IncrementalID())
	sha := d.cicd().GitSHA()
	if sha == "" {
		sha = d.git().GitSHA()
	}
	sha = trimLen(sha, 7)
	return d.SanitizeTag(fmt.Sprintf("%s-%s-%s", branch, id, sha))
}

func (d *Docker) BuildxCacheFrom() string {
	return d.Env.GetDefault("DOCKER_BUILDX_FROM", "/tmp/.buildx-cache")
}

func (d *Docker) BuildxCacheTo() string {
	return d.Env.GetDefault("DOCKER_BUILDX_TO", "/tmp/.buildx-cache-new")
}

// Build a docker image using buildx
func (d *Docker) Build(ctx context.Context) error {
	image := d.Image()
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
	return pipe.NewPiped("docker", args...).Run(ctx)
}

func (d *Docker) Lint(ctx context.Context) error {
	allDocker, err := files.AllWithExtension("Dockerfile")
	if err != nil {
		return err
	}
	if len(allDocker) == 0 {
		fmt.Println("No Dockerfiles to lint")
		return nil
	}
	var hadoErr error
	for _, d := range allDocker {
		if err := func() error {
			f, err := os.Open(d)
			defer func() {
				if err := f.Close(); err != nil {
					fmt.Println("unable to fully close file")
				}
			}()
			if err != nil {
				return fmt.Errorf("uanble to open dockerfile for reading: %w", err)
			}
			if err := pipe.NewPiped("docker", `run`, `-i`, `--rm`, `hadolint/hadolint`).Execute(ctx, f, os.Stdout, os.Stderr); err != nil {
				hadoErr = err
			}
			return nil
		}(); err != nil {
			hadoErr = err
		}
	}
	return hadoErr
}

func (d *Docker) RotateCache(ctx context.Context) error {
	from := d.BuildxCacheFrom()
	to := d.BuildxCacheTo()
	if files.IsDir(from) {
		if err := os.RemoveAll(from); err != nil {
			return err
		}
	} else {
		fmt.Printf("no from directory to remove: %s\n", from)
	}
	if files.IsDir(to) {
		return os.Rename(to, from)
	} else {
		fmt.Printf("no to directory to rename: %s\n", to)
	}
	return nil
}

// Lint a dockerfile using hadolint
func Lint(ctx context.Context) error {
	return Instance.Lint(ctx)
}

// Build the docker image
func Build(ctx context.Context) error {
	return Instance.Build(ctx)
}

// RotateCache will rotate the FROM and TO docker buildx caches.
func RotateCache(ctx context.Context) error {
	return Instance.RotateCache(ctx)
}
