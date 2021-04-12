package docker

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cresta/magehelper/cicd"
	"github.com/cresta/magehelper/docker/registry"
	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/errhelp"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/git"
	"github.com/cresta/magehelper/pipe"
)

func trimLen(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[0:maxLen]
}

var Instance = &Docker{}

type Docker struct {
	Env      env.Env
	Registry registry.Registry
	CiCd     cicd.CiCd
	Git      *git.Git
}

func (d *Docker) registry() registry.Registry {
	if d.Registry == nil {
		return registry.Instance
	}
	return d.Registry
}

func (d *Docker) cicd() cicd.CiCd {
	if d.CiCd == nil {
		return cicd.Instance()
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
	return d.ImageWithTag(d.Tag())
}

func (d *Docker) ImageWithTag(tag string) string {
	reg := d.registry().ContainerRegistry()
	if reg != "" {
		reg += "/"
	}
	return fmt.Sprintf("%s%s:%s", reg, d.Repository(), tag)
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

func (d *Docker) RecordImage() error {
	fileName := d.Env.Get("DOCKER_IMAGE_FILE")
	image := Instance.Image()
	if fileName == "" || fileName == "-" {
		return errhelp.SecondErr(fmt.Println(image))
	}
	return os.WriteFile(fileName, []byte(image), 0600)
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

// If DOCKER_LATEST_BRANCH is set, any pushes to that branch will also get a "latest" tag built and pushed
func (d *Docker) alsoTagLatest() bool {
	latestBranchName := d.Env.Get("DOCKER_LATEST_BRANCH")
	if latestBranchName == "" {
		return false
	}
	ref := d.cicd().GitRef()
	if ref == "" {
		ref = d.git().GitRef()
	}
	return d.git().BranchName(ref) == latestBranchName
}

// Build a docker image using buildx
func (d *Docker) Build(ctx context.Context) error {
	return d.BuildWithConfig(ctx, BuildConfig{})
}

type BuildConfig struct {
	BuildArgs []string
}

// Build a docker image using buildx
func (d *Docker) BuildWithConfig(ctx context.Context, config BuildConfig) error {
	image := d.Image()
	args := []string{"buildx", "build"}
	if d.Env.Get("DOCKER_PUSH") == "true" {
		args = append(args, "--push")
	} else {
		args = append(args, "--load")
	}
	for _, a := range config.BuildArgs {
		args = append(args, "--build-arg", a)
	}
	if d.alsoTagLatest() {
		args = append(args, "-t", d.ImageWithTag("latest"))
	}
	cacheFrom := d.BuildxCacheFrom()
	cacheTo := d.BuildxCacheTo()
	if files.IsDir(cacheFrom) {
		args = append(args, fmt.Sprintf("--cache-from=type=local,src=%s", cacheFrom))
	}
	if files.IsDir(cacheTo) {
		args = append(args, fmt.Sprintf("--cache-from=type=local,src=%s", cacheTo))
	}
	if df := d.Env.Get("DOCKER_FILE"); df != "" {
		args = append(args, "-f", df)
	}
	args = append(args, fmt.Sprintf("--cache-to=type=local,dest=%s", cacheTo), "-t", image, d.Env.GetDefault("DOCKER_BUILD_ROOT", "."))
	if err := pipe.NewPiped("docker", args...).Run(ctx); err != nil {
		return err
	}
	fmt.Println("Build docker image:", image)
	return nil
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
	to := d.BuildxCacheTo()
	if !files.IsDir(to) {
		fmt.Printf("no to directory to rotate into new: %s\n", to)
		return nil
	}
	from := d.BuildxCacheFrom()
	if files.IsDir(from) {
		fmt.Printf("removing directory %s\n", from)
		if err := os.RemoveAll(from); err != nil {
			return err
		}
	} else {
		fmt.Printf("no from directory to remove: %s\n", from)
	}
	fmt.Printf("renaming %s -> %s\n", to, from)
	return os.Rename(to, from)
}

// Use Hadolint to run a lint against all dockerfiles in this repository
func Lint(ctx context.Context) error {
	return Instance.Lint(ctx)
}

// Run a buildx docker build
func Build(ctx context.Context) error {
	return Instance.Build(ctx)
}

// Rotate the buildx caches for github
func RotateCache(ctx context.Context) error {
	return Instance.RotateCache(ctx)
}

// Record the image to a file (defined by DOCKER_IMAGE_FILE) or to stdout
func RecordImage(ctx context.Context) error {
	return Instance.RecordImage()
}
