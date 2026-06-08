package registry

import (
	"context"

	"github.com/opencontainers/go-digest"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type DeleteBlob struct {
	endpointregistryv2.DeleteBlob

	namespace content.Namespace `inject:""`
}

func (req *DeleteBlob) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}

	err = blobs.Remove(ctx, digest.Digest(req.Digest))
	if err != nil {
		return nil, err
	}
	return nil, nil
}
