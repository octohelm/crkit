package fs

import (
	"context"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/content"
)

type repository struct {
	workspace *workspace
	named     reference.Named
}

func (r *repository) Named() reference.Named {
	return r.named
}

func (r *repository) Blobs(ctx context.Context) (content.BlobStore, error) {
	return newLinkedBlobStore(r.workspace, r.named), nil
}

func (r *repository) Manifests(ctx context.Context) (content.ManifestService, error) {
	return &manifestService{blobStore: newLinkedBlobStoreAsManifestService(r.workspace, r.named)}, nil
}

func (r *repository) Tags(ctx context.Context) (content.TagService, error) {
	return &tagService{
		named:           r.named,
		workspace:       r.workspace,
		manifestService: &manifestService{blobStore: newLinkedBlobStoreAsManifestService(r.workspace, r.named)},
	}, nil
}
