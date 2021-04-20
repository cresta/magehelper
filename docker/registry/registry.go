package registry

import "context"

type Registry interface {
	ContainerRegistry() string
	Login(ctx context.Context) error
}

// Instance is where you push docker images
var Instance Registry

// CacheInstance is where you push the caches for docker images.  Libraries should default to Instance if this is null
var CacheInstance Registry

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
