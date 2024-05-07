package registry

import (
	"context"
	"github.com/distribution/distribution/v3"
	"github.com/distribution/reference"
)

type namespace struct {
	distribution.Namespace
	baseHost BaseHost
}

func (n *namespace) Repository(ctx context.Context, named reference.Named) (distribution.Repository, error) {
	return n.Namespace.Repository(ctx, n.baseHost.TrimNamed(named))
}
