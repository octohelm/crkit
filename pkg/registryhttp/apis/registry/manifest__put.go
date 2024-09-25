package registry

import (
	"context"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
	registryoperator "github.com/octohelm/crkit/pkg/registryhttp/apis/registry/operator"
)

func (PutManifest) MiddleOperators() courier.MiddleOperators {
	return courier.MiddleOperators{
		&registryoperator.NameScoped{},
	}
}

type PutManifest struct {
	courierhttp.MethodPut `path:"/manifests/{reference}"`

	Reference content.TagOrDigest `name:"reference" in:"path"`
	Manifest  manifestv1.Payload  `in:"body"`
}

func (req *PutManifest) Output(ctx context.Context) (any, error) {
	repo := content.RepositoryContext.From(ctx)

	manifests, err := repo.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	d, err := manifests.Put(ctx, req.Manifest)
	if err != nil {
		return nil, err
	}

	if tag, err := req.Reference.Tag(); err == nil {
		tags, err := repo.Tags(ctx)
		if err != nil {
			return nil, err
		}

		if err := tags.Tag(ctx, tag, manifestv1.Descriptor{
			Digest: d,
		}); err != nil {
			return nil, err
		}
	}

	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(200),
		courierhttp.WithMetadata("Docker-Content-Digest", d.String()),
	), nil
}
