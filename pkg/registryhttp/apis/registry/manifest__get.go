package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type GetManifest struct {
	endpointregistryv2.GetManifest

	namespace content.Namespace `inject:""`
}

func (req *GetManifest) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

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

	p, err := manifestv1.From(m)
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap(
		p,
		courierhttp.WithMetadata("Docker-Content-Digest", string(dgst)),
		courierhttp.WithMetadata("Content-Type", m.Type()),
	), nil
}
