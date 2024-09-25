package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
	registryoperator "github.com/octohelm/crkit/pkg/registryhttp/apis/registry/operator"
	"github.com/opencontainers/go-digest"
)

func (GetBlob) MiddleOperators() courier.MiddleOperators {
	return courier.MiddleOperators{
		&registryoperator.NameScoped{},
	}
}

type GetBlob struct {
	courierhttp.MethodGet `path:"/blobs/{digest}"`

	Digest content.Digest `name:"digest" in:"path"`
}

func (req *GetBlob) Output(ctx context.Context) (any, error) {
	repo := content.RepositoryContext.From(ctx)

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
