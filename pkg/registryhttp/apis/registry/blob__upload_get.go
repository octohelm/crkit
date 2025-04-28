package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/octohelm/courier/pkg/courierhttp"
)

// +gengo:injectable
type GetUploadBlob struct {
	courierhttp.MethodGet `path:"/{name...}/blobs/uploads/{id}"`
	NameScoped

	ID string `name:"id" in:"path"`
}

func (req *GetUploadBlob) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
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

	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(http.StatusAccepted),
		courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo.Named().Name(), w.ID())),
		courierhttp.WithMetadata("Range", fmt.Sprintf("0-%d", endRange)),
	), nil
}
