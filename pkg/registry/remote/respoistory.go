package remote

import (
	"context"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type repository struct {
	namespace *namespace

	named reference.Named
	repo  name.Repository

	pusher *remote.Pusher
	puller *remote.Puller
}

func (r *repository) Named() reference.Named {
	return r.named
}

func (r *repository) Manifests(ctx context.Context, options ...distribution.ManifestServiceOption) (distribution.ManifestService, error) {
	return &manifestService{repository: r}, nil
}

func (r *repository) Blobs(ctx context.Context) distribution.BlobStore {
	return &blobStore{repository: r}
}

func (r *repository) Tags(ctx context.Context) distribution.TagService {
	return &tagService{repository: r}
}
