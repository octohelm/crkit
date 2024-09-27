package content

import (
	"context"

	"github.com/distribution/reference"
)

// +gengo:injectable:provider
type Repository interface {
	Named() reference.Named
	Manifests(ctx context.Context) (ManifestService, error)
	Tags(ctx context.Context) (TagService, error)
	Blobs(ctx context.Context) (BlobStore, error)
}
