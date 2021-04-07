package registry

import "context"

type Registry interface {
	ContainerRegistry() string
	Login(ctx context.Context) error
}
