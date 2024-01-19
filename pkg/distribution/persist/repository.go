package persist

import (
	"github.com/octohelm/crkit/pkg/distribution"
	"github.com/octohelm/unifs/pkg/filesystem"
)

func NewRepository(fsys filesystem.FileSystem, name string) distribution.Repository {
	b := &blobService{fsys: fsys}

	return &repository{
		blobs: b,
	}
}

type repository struct {
	blobs     distribution.BlobService
	manifests distribution.ManifestService
	tags      distribution.TagService
}

func (r *repository) Blobs() distribution.BlobService {
	return r.blobs
}

func (r *repository) Manifests() distribution.ManifestService {
	return r.manifests
}

func (r *repository) Tags() distribution.TagService {
	return r.tags
}
