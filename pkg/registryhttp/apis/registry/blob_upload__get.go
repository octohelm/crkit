package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/octohelm/courier/pkg/courierhttp"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type GetBlobUpload struct {
	endpointregistryv2.GetBlobUpload

	namespace content.Namespace `inject:""`
}

func (req *GetBlobUpload) Output(ctx context.Context) (any, error) {
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
	defer w.Close()

	endRange := w.Size(ctx)
	if endRange > 0 {
		endRange = endRange - 1
	}

	return courierhttp.Wrap[any](
		nil,
		courierhttp.WithStatusCode(http.StatusAccepted),
		courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo.Named().Name(), w.ID())),
		courierhttp.WithMetadata("Range", fmt.Sprintf("0-%d", endRange)),
	), nil
}
