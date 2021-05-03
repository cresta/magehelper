package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/magefile/mage/mg"
	"os"
	"path/filepath"
	"strings"

	"github.com/cresta/magehelper/env"
	"github.com/cresta/magehelper/files"
	"github.com/cresta/magehelper/pipe"
)

var Instance = &Helm{}

type Helm struct {
	Env *env.Env
}

func (h *Helm) s3individualChartPrefix() string {
	return h.Env.Get("HELM_S3_PREFIX")
}

func (h *Helm) repoNamePrefix() string {
	return h.Env.Get("HELM_REPO_NAME_PREFIX")
}

func (h *Helm) staticRepos() []string {
	return strings.Split(h.Env.Get("HELM_STATIC_REPOS"), ";")
}

func (h *Helm) initS3Repo() bool {
	return h.Env.GetDefault("HELM_S3_INIT", "false") == "true"
}

func (h *Helm) BuildLint(ctx context.Context) error {
	validCharts, err := h.listValidCharts(ctx)
	if err != nil {
		return err
	}
	if !files.IsDir("check-templates") {
		if err := os.Mkdir("check-templates", 0755); err != nil {
			return fmt.Errorf("unable to make check directory: %w", err)
		}
	}
	for _, c := range validCharts {
		existingTgz, err := files.AllWithExtensionExactlyInDir("tgz", filepath.Join("charts", c))
		if err != nil {
			return fmt.Errorf("unable to find existing tgz: %w", err)
		}
		if len(existingTgz) != 0 {
			return fmt.Errorf("found an existing tgz file.  You should probably delete this: %s", strings.Join(existingTgz, ","))
		}
		if err := pipe.Shell("helm dep build --skip-refresh").WithDir(filepath.Join("charts", c)).Run(ctx); err != nil {
			return err
		}
		if err := pipe.Shell("helm lint").WithDir(filepath.Join("charts", c)).Run(ctx); err != nil {
			return err
		}
		if err := pipe.Shell("helm package .").WithDir(filepath.Join("charts", c)).Run(ctx); err != nil {
			return err
		}
		templateOutput, err := os.Create(filepath.Join("check-templates", c+".yaml"))
		if err != nil {
			return fmt.Errorf("unable to open check template output: %w", err)
		}
		if err := pipe.Shell("helm template .").WithDir(filepath.Join("charts", c)).Execute(ctx, nil, templateOutput, nil); err != nil {
			return err
		}
		if err := templateOutput.Close(); err != nil {
			return fmt.Errorf("unable to close template file: %w", err)
		}
	}
	return nil
}

func (h *Helm) PushRepos(ctx context.Context) error {
	validCharts, err := h.listValidCharts(ctx)
	if err != nil {
		return err
	}
	for _, c := range validCharts {
		existingTgz, err := files.AllWithExtensionExactlyInDir(".tgz", filepath.Join("charts", c))
		if err != nil {
			return fmt.Errorf("unable to find existing tgz: %w", err)
		}
		if len(existingTgz) != 1 {
			fmt.Println("unable to find a built package for chart ", c)
			continue
		}
		if err := pipe.NewPiped("helm", "s3", "push", "--ignore-if-exists", filepath.Join("charts", c, existingTgz[0]), h.repoNameForChart(c)).Run(ctx); err != nil {
			return fmt.Errorf("unable to push helm chart: %w", err)
		}
	}
	return nil
}

func (h *Helm) repoNameForChart(s string) string {
	return h.s3individualChartPrefix() + s
}

func (h *Helm) S3Setup(ctx context.Context) error {
	if !strings.HasPrefix(h.s3individualChartPrefix(), `s3:\\`) {
		return fmt.Errorf("no S3 prefix for chart repo.  Maybe not s3")
	}
	// Install s3 plugin
	hasP, err := h.hasPlugin(ctx, "s3")
	if err != nil {
		return err
	}
	if !hasP {
		if err := pipe.Shell("helm plugin install https://github.com/hypnoglow/helm-s3.git").Run(ctx); err != nil {
			return err
		}
	}
	if !files.IsDir("./charts") {
		fmt.Println("No charts directory.  Skipping S3 setup")
		return nil
	}

	repos, err := h.listRepos(ctx)
	if err != nil {
		return err
	}

	validCharts, err := h.listValidCharts(ctx)
	if err != nil {
		return err
	}
	for _, c := range validCharts {
		repoName := h.repoNamePrefix() + c
		if containsRepo(repos, repoName) {
			continue
		}
		if err := pipe.NewPiped("helm", "repo", "add", repoName, h.repoNameForChart(c)).Run(ctx); err != nil {
			return err
		}
		if h.initS3Repo() {
			if err := pipe.NewPiped("helm", "s3", "init", h.repoNameForChart(c)).Run(ctx); err != nil {
				fmt.Printf("uanble to init s3 repo.  This is sometimes OK if the repo is already init: %v\n", err)
			}
		}
	}
	for _, s := range h.staticRepos() {
		parts := strings.Split(s, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid ENV variable HELM_STATIC_REPOS")
		}
		if containsRepo(repos, parts[0]) {
			continue
		}
		if err := pipe.NewPiped("helm", "repo", "add", parts[0], parts[1]).Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

type Repo struct {
	Name string `json:"name"`
}

func containsRepo(set []Repo, s string) bool {
	for _, k := range set {
		if k.Name == s {
			return true
		}
	}
	return false
}

func (h *Helm) listRepos(ctx context.Context) ([]Repo, error) {
	var buf bytes.Buffer
	if err := pipe.Shell("helm repo list -o json").Execute(ctx, nil, &buf, nil); err != nil {
		// helm returns exit code 1 if there are no repos
		return nil, nil
	}
	var ret []Repo
	if err := json.Unmarshal(buf.Bytes(), &ret); err != nil {
		return nil, fmt.Errorf("did not see valid JSON on repo list: %w", err)
	}
	return ret, nil
}

func (h *Helm) listValidCharts(ctx context.Context) ([]string, error) {
	if !files.IsDir("./charts") {
		if mg.Verbose() {
			fmt.Println("unable to find any charts")
		}
		return nil, nil
	}
	// Find all charts in the charts directory
	entries, err := os.ReadDir("./charts")
	if err != nil {
		return nil, fmt.Errorf("unable to list charts directory: %w", err)
	}
	ret := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == "." || e.Name() == ".." {
			continue
		}
		path := filepath.Join("charts", e.Name(), "Chart.yaml")
		if mg.Verbose() {
			fmt.Println("Looking for chart at", path)
		}

		// Go into this directory and set it up
		if !files.FileExists(path) {
			continue
		}
		ret = append(ret, e.Name())
	}
	return ret, nil
}

func (h *Helm) hasPlugin(ctx context.Context, name string) (bool, error) {
	p, err := h.listPlugins(ctx)
	if err != nil {
		return false, err
	}
	for _, s := range p {
		if s == name {
			return true, nil
		}
	}
	return false, nil
}

func (h *Helm) listPlugins(ctx context.Context) ([]string, error) {
	var buf bytes.Buffer
	if err := pipe.Shell("helm plugin list").Execute(ctx, nil, &buf, nil); err != nil {
		return nil, fmt.Errorf("unable to list plugins: %w", err)
	}
	lines := strings.Split(buf.String(), "\n")
	var ret []string
	for idx, line := range lines {
		if idx == 0 {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) == 0 {
			continue
		}
		p1 := strings.TrimSpace(parts[0])
		ret = append(ret, p1)
	}
	return ret, nil
}

// S3Setup will create the helm repos that point to S3
func S3Setup(ctx context.Context) error {
	return Instance.S3Setup(ctx)
}

// BuildLint runs a basic build/lint for helm
func BuildLint(ctx context.Context) error {
	return Instance.BuildLint(ctx)
}

// PushRepos will push previously built charts
func PushRepos(ctx context.Context) error {
	return Instance.PushRepos(ctx)
}
