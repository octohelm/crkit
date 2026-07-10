package main

import (
	"context"
	"dagger/crkit/internal/dagger"
)

type Crkit struct{}

func (t *Crkit) Service(
	ctx context.Context,
	// +defaultPath="/"
	// +ignore=["target"]
	source *dagger.Directory,
) (*dagger.Service, error) {
	ctr, err := t.Container(ctx, source, "").Build(ctx, "", "")
	if err != nil {
		return nil, err
	}

	return ctr.
		WithMountedCache("/etc/registry", dag.CacheVolume("registry")).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true}), nil
}
