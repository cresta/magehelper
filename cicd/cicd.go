package cicd

import (
	"github.com/cresta/magehelper/env"
)

type registry struct {
	constructors []Constructor
}

var globalRegistry registry

type Constructor func() (CiCd, error)

func Register(constructor func() (CiCd, error)) {
	globalRegistry.constructors = append(globalRegistry.constructors, constructor)
}

var Instance CiCd

func init() {
	var err error
	Instance, err = Create()
	if err != nil {
		panic(err)
	}
}

func Create() (CiCd, error) {
	for _, c := range globalRegistry.constructors {
		ci, err := c()
		if err != nil {
			return nil, err
		}
		if ci != nil {
			return ci, nil
		}
	}
	return &Local{}, nil
}

func MustCreate() CiCd {
	c, err := Create()
	if err != nil {
		panic(err)
	}
	return c
}

type CiCd interface {
	IncrementalID() string
	GitRef() string
	GitSHA() string
	GitRepository() string
	Name() string
}

type Local struct {
	Env *env.Env
}

func (l *Local) IncrementalID() string {
	return l.Env.GetDefault("BUILD_ID", "0")
}

func (l *Local) GitRef() string {
	return ""
}

func (l *Local) GitSHA() string {
	return ""
}

func (l *Local) Name() string {
	return "local"
}

func (l *Local) GitRepository() string {
	return ""
}

var _ CiCd = &Local{}
