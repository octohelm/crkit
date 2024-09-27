package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/uploadcache"
)

// +gengo:injectable
type CancelUploadBlob struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/uploads/{id}"`
	NameScoped

	ID string `name:"id" in:"path"`

	uploadCache uploadcache.UploadCache `inject:""`
}

func (req *CancelUploadBlob) Output(ctx context.Context) (any, error) {
	w, err := req.uploadCache.Resume(ctx, req.ID)
	if err != nil {
		return nil, nil
	}
	_ = w.Close()
	return nil, nil
}
