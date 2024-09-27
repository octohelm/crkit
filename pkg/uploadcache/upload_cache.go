package uploadcache

import (
	"context"

	"github.com/octohelm/crkit/pkg/content"
)

// +gengo:injectable:provider
type UploadCache interface {
	BlobWriter(ctx context.Context, repo content.Repository) (content.BlobWriter, error)
	Resume(ctx context.Context, id string) (content.BlobWriter, error)
}
