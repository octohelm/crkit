package registry

import (
	"context"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

type GetBlob struct {
	courierhttp.MethodGet `path:"/{name...}/blobs/{digest}"`

	NameScoped

	Digest content.Digest `name:"digest" in:"path"`
}

func (req *GetBlob) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
	if err != nil {
		return nil, err
	}

	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}

	b, err := blobs.Open(ctx, digest.Digest(req.Digest))
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap(
		b,
		courierhttp.WithMetadata("Docker-Content-Digest", string(req.Digest)),
	), nil
}
