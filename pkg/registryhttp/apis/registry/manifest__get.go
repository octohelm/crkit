package registry

import (
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
	registryoperator "github.com/octohelm/crkit/pkg/registryhttp/apis/registry/operator"
)

import (
	"context"
)

func (GetManifest) MiddleOperators() courier.MiddleOperators {
	return courier.MiddleOperators{
		&registryoperator.NameScoped{},
	}
}

type GetManifest struct {
	courierhttp.MethodGet `path:"/manifests/{reference}"`

	Reference content.TagOrDigest `name:"reference" in:"path"`
}

func (req *GetManifest) Output(ctx context.Context) (any, error) {
	repo := content.RepositoryContext.From(ctx)

	dgst, err := req.Reference.Digest()
	if err != nil {
		tags, err := repo.Tags(ctx)
		if err != nil {
			return nil, err
		}

		d, err := tags.Get(ctx, string(req.Reference))
		if err != nil {
			return nil, err
		}
		dgst = d.Digest
	}

	manifests, err := repo.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	m, err := manifests.Get(ctx, dgst)
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap(m,
		courierhttp.WithMetadata("Docker-Content-Digest", dgst.String()),
		courierhttp.WithMetadata("Content-Type", m.Type()),
	), nil
}
