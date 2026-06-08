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
type PutManifest struct {
	endpointregistryv2.PutManifest

	namespace content.Namespace `inject:""`
}

func (req *PutManifest) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

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

	return courierhttp.Wrap[any](
		nil,
		courierhttp.WithStatusCode(201),
		courierhttp.WithMetadata("Docker-Content-Digest", d.String()),
	), nil
}
