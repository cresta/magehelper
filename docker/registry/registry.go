package registry

import "context"

type Registry interface {
	ContainerRegistry() string
	Login(ctx context.Context) error
}

var Instance Registry

func init() {
	Instance = &Local{}
}

type Local struct {
}

func (l *Local) ContainerRegistry() string {
	return ""
}

func (l *Local) Login(ctx context.Context) error {
	return nil
}

var _ Registry = &Local{}
