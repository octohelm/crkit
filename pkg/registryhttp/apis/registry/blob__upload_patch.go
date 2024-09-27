package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/uploadcache"
)

// +gengo:injectable
type UploadPatchBlob struct {
	courierhttp.MethodPatch `path:"/{name...}/blobs/uploads/{id}"`

	NameScoped

	ID string `name:"id" in:"path"`

	Chunk io.ReadCloser `in:"body"`

	uploadCache uploadcache.UploadCache `inject:""`
}

func (req *UploadPatchBlob) Output(ctx context.Context) (any, error) {
	defer req.Chunk.Close()

	repo, err := req.Repository(ctx)
	if err != nil {
		return nil, err
	}

	w, err := req.uploadCache.Resume(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(w, req.Chunk); err != nil {
		return nil, err
	}

	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(http.StatusAccepted),
		courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo.Named().Name(), w.ID())),
	), nil
}
