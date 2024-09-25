package fs

import (
	"context"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"
)

type repository struct {
	named reference.Named
	fs    filesystem.FileSystem
}

func (r *repository) Named() reference.Named {
	return r.named
}

func (r *repository) Blobs(ctx context.Context) (content.BlobStore, error) {
	return &linkedBlobStore{
		named:        r.named,
		fs:           r.fs,
		blobStore:    NewBlobStore(r.fs),
		linkPathFunc: defaultLayout.RepositoryLayerLinkPath,
		errUnknownFunc: func(named reference.Named, dgst digest.Digest) error {
			return &content.ErrManifestBlobUnknown{
				Digest: dgst,
			}
		},
	}, nil
}

func (r *repository) Manifests(ctx context.Context) (content.ManifestService, error) {
	return &manifestService{
		named: r.named,
		fs:    r.fs,
		blobStore: &linkedBlobStore{
			named:        r.named,
			fs:           r.fs,
			blobStore:    NewBlobStore(r.fs),
			linkPathFunc: defaultLayout.RepositoryManifestRevisionLinkPath,
			errUnknownFunc: func(named reference.Named, dgst digest.Digest) error {
				return &content.ErrManifestUnknownRevision{
					Name:     named.Name(),
					Revision: dgst,
				}
			},
		},
	}, nil
}

func (r *repository) Tags(ctx context.Context) (content.TagService, error) {
	ms, err := r.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	return &tagService{
		named:           r.named,
		fs:              r.fs,
		manifestService: ms,
	}, err
}
