package git

import (
	"strings"

	"github.com/magefile/mage/sh"
)

type Git struct{}

var Instance Git

func (g *Git) GitRef() string {
	s, err := sh.Output("git", "symbolic-ref", "HEAD")
	if err == nil {
		return s
	}
	return ""
}

func (g *Git) BranchName(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ref
}

func (g *Git) TagName(ref string) string {
	if strings.HasPrefix(ref, "refs/tags/") {
		return strings.TrimPrefix(ref, "refs/tags/")
	}
	return ""
}

func (g *Git) GitSHA() string {
	s, err := sh.Output("git", "rev-parse", "--verify", "HEAD")
	if err == nil {
		return s
	}
	return ""
}

func (g *Git) RemoteRepository() string {
	s, err := sh.Output("git", "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}
	// S is something like
	//  git@github.com:cresta/project.git
	//  https://github.com/nginx/nginx.git
	s = strings.TrimSuffix(s, ".git")
	if strings.HasPrefix(s, "git@github.com:") {
		return strings.TrimPrefix(s, "git@github.com:")
	}
	if strings.HasPrefix(s, "https://github.com/") {
		return strings.TrimPrefix(s, "https://github.com/")
	}
	return ""
}
