package registry

import (
	"context"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type DeleteManifest struct {
	endpointregistryv2.DeleteManifest

	namespace content.Namespace `inject:""`
}

func (req *DeleteManifest) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

	dgst, err := req.Reference.Digest()
	if err == nil {
		manifests, err := repo.Manifests(ctx)
		if err != nil {
			return nil, err
		}

		if err := manifests.Delete(ctx, dgst); err != nil {
			return nil, err
		}

		return nil, nil
	}

	tag := string(req.Reference)
	tags, err := repo.Tags(ctx)
	if err != nil {
		return nil, err
	}
	if err := tags.Untag(ctx, tag); err != nil {
		return nil, err
	}
	return nil, nil
}
