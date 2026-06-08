package registry

import (
	"context"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type CancelBlobUpload struct {
	endpointregistryv2.CancelBlobUpload

	namespace content.Namespace `inject:""`
}

func (req *CancelBlobUpload) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}

	w, err := blobs.Resume(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if err := w.Cancel(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}
