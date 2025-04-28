package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"
)

// +gengo:injectable
type CancelUploadBlob struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/uploads/{id}"`
	NameScoped

	ID string `name:"id" in:"path"`
}

func (req *CancelUploadBlob) Output(ctx context.Context) (any, error) {
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

	if err := w.Cancel(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}
