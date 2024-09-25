package content

import (
	"context"

	"github.com/distribution/reference"
	contextx "github.com/octohelm/x/context"
)

var RepositoryContext = contextx.New[Repository]()

type Repository interface {
	Named() reference.Named
	Manifests(ctx context.Context) (ManifestService, error)
	Tags(ctx context.Context) (TagService, error)
	Blobs(ctx context.Context) (BlobStore, error)
}
