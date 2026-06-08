package registry

import (
	"context"
	"fmt"

	"github.com/octohelm/courier/pkg/courierhttp"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type HeadManifest struct {
	endpointregistryv2.HeadManifest

	namespace content.Namespace `inject:""`
}

func (req *HeadManifest) Output(ctx context.Context) (x any, e error) {
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

	desc, err := manifests.Info(ctx, dgst)
	if err != nil {
		return nil, err
	}

	m, err := manifests.Get(ctx, dgst)
	if err != nil {
		return nil, err
	}

	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#checking-if-content-exists-in-the-registry
	return courierhttp.Wrap[any](
		nil,
		courierhttp.WithStatusCode(200),
		courierhttp.WithMetadata("Docker-Content-Digest", desc.Digest.String()),
		courierhttp.WithMetadata("Content-Length", fmt.Sprintf("%d", desc.Size)),
		courierhttp.WithMetadata("Content-Type", m.Type()),
	), nil
}
