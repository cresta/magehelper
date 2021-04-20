package docker

import (
	"context"
	"fmt"
	"os"
	"regexp"

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
	Env             env.Env
	Registry        registry.Registry
	CacheRegistry   registry.Registry
	CiCd            cicd.CiCd
	Git             *git.Git
	IgnoreFastBuild bool
}

func (d *Docker) registry() registry.Registry {
	if d.Registry == nil {
		return registry.Instance
	}
	return d.Registry
}

func (d *Docker) cacheRegistry() registry.Registry {
	if d.CacheRegistry != nil {
		return d.CacheRegistry
	}
	if registry.CacheInstance != nil {
		return registry.CacheInstance
	}
	return d.registry()
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
	return d.ImageWithTagForRegistry(d.registry(), tag)
}

func (d *Docker) ImageWithTagForRegistry(regist registry.Registry, tag string) string {
	reg := regist.ContainerRegistry()
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

func (d *Docker) CacheRepository() string {
	if d.Env.Get("DOCKER_CACHE_REPOSITORY") != "" {
		return d.Env.Get("DOCKER_REPOSITORY")
	}
	return d.Repository()
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
	if tagName := d.tagName(); tagName != "" {
		// Note: Should upgrade this to only happen if it matches the regex v[0-9]+
		if len(tagName) > 0 && tagName[0] == 'v' {
			tagName = tagName[1:]
		}
		return d.SanitizeTag(tagName)
	}
	// 128 max characters.  Reserve 64 for the branch name to give room for the rest
	branch := trimLen(d.branchName(), 64)
	id := fmt.Sprintf("%s.%s", d.cicd().Name(), d.cicd().IncrementalID())
	sha := d.cicd().GitSHA()
	if sha == "" {
		sha = d.git().GitSHA()
	}
	sha = trimLen(sha, 7)
	return d.SanitizeTag(fmt.Sprintf("%s-%s-%s", branch, id, sha))
}

func (d *Docker) latestBranch() string {
	return d.Env.GetDefault("DOCKER_LATEST_BRANCH", "master")
}

func (d *Docker) remoteCacheTags(forceLatest bool) []string {
	// build args for --cache-to= for a remote
	var cacheToTags []string
	branchName := d.branchName()
	if forceLatest || branchName == d.latestBranch() {
		cacheToTags = append(cacheToTags, "latest")
	}
	if branchName != "" && branchName != "latest" {
		cacheToTags = append(cacheToTags, branchName)
	}
	ret := make([]string, 0, len(cacheToTags))
	// Turn them into sanitized tags
	// This format allows reusing the cache repository
	for _, cacheToTag := range cacheToTags {
		cacheTag := d.SanitizeTag(fmt.Sprintf("cache-%s-%s", d.CacheRepository(), cacheToTag))
		ret = append(ret, d.ImageWithTagForRegistry(d.cacheRegistry(), cacheTag))
	}
	return ret
}

func (d *Docker) remoteCacheFrom() []string {
	cacheFromTags := d.remoteCacheTags(true)
	ret := make([]string, 0, len(cacheFromTags))
	// Turn them into sanitized tags
	for _, cacheToTag := range cacheFromTags {
		ret = append(ret, fmt.Sprintf("--cache-from=%s", cacheToTag))
	}
	return ret
}

func (d *Docker) remoteCacheTo() []string {
	cacheToTags := d.remoteCacheTags(false)
	ret := make([]string, 0, len(cacheToTags))
	// Turn them into sanitized tags
	for _, cacheToTag := range cacheToTags {
		ret = append(ret, fmt.Sprintf("--cache-to=type=registry,ref=%s,mode=max", cacheToTag))
	}
	return ret
}

func (d *Docker) branchName() string {
	ref := d.cicd().GitRef()
	if ref == "" {
		ref = d.git().GitRef()
	}
	return d.git().BranchName(ref)
}

func (d *Docker) tagName() string {
	ref := d.cicd().GitRef()
	if ref == "" {
		ref = d.git().GitRef()
	}
	return d.git().TagName(ref)
}

func (d *Docker) BuildxCacheFrom() string {
	return d.Env.GetDefault("DOCKER_BUILDX_FROM", "/tmp/.buildx-cache")
}

func (d *Docker) BuildxCacheTo() string {
	return d.Env.GetDefault("DOCKER_BUILDX_TO", "/tmp/.buildx-cache-new")
}

// If DOCKER_MUTABLE_TAGS is true, then we also build mutable tags (tags that are likely to be overridden)
func (d *Docker) mutableBuildTags() []string {
	if d.Env.Get("DOCKER_MUTABLE_TAGS") != "true" {
		return nil
	}
	branchName := d.branchName()
	ret := []string{branchName}
	if branchName == d.latestBranch() {
		ret = append(ret, "latest")
	}
	return ret
}

// Build a docker image using buildx
func (d *Docker) Build(ctx context.Context) error {
	return d.BuildWithConfig(ctx, BuildConfig{})
}

type BuildConfig struct {
	BuildArgs []string
}

func (d *Docker) ImageExists(ctx context.Context, tag string) bool {
	err := pipe.Shell("docker inspect --type=image "+tag).Execute(ctx, nil, nil, nil)
	return err == nil
}

// Build a docker image using buildx
func (d *Docker) BuildWithConfig(ctx context.Context, config BuildConfig) error {
	pushBuiltImage := d.Env.Get("DOCKER_PUSH") == "true"
	pushRemoteCache := d.Env.Get("DOCKER_REMOTE_CACHE") == "true"
	pushLocalCache := !pushRemoteCache
	image := d.Image()
	args := []string{"buildx", "build"}
	if pushBuiltImage {
		args = append(args, "--push")
	} else {
		args = append(args, "--load")
	}
	for _, a := range config.BuildArgs {
		args = append(args, "--build-arg", a)
	}
	for _, mutableTag := range d.mutableBuildTags() {
		args = append(args, "-t", d.ImageWithTag(mutableTag))
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
	args = append(args, d.remoteCacheFrom()...)
	if pushRemoteCache {
		// Two remote caches are "latest" and the branch name
		// Push this cache to the branch name, and also latest if we're on the main branch
		args = append(args, d.remoteCacheTo()...)
	}
	if pushLocalCache {
		// Use local cache
		args = append(args, fmt.Sprintf("--cache-to=type=local,dest=%s", cacheTo))
	}
	args = append(args, "-t", image, d.Env.GetDefault("DOCKER_BUILD_ROOT", "."))
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
