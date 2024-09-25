package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
	registryoperator "github.com/octohelm/crkit/pkg/registryhttp/apis/registry/operator"
	"github.com/octohelm/crkit/pkg/uploadcache"
)

func (UploadPatchBlob) MiddleOperators() courier.MiddleOperators {
	return courier.MiddleOperators{
		&registryoperator.NameScoped{},
	}
}

type UploadPatchBlob struct {
	courierhttp.MethodPatch `path:"/blobs/uploads/{id}"`
	ID                      string        `name:"id" in:"path"`
	Chunk                   io.ReadCloser `in:"body"`
}

func (req *UploadPatchBlob) Output(ctx context.Context) (any, error) {
	defer req.Chunk.Close()

	repo := content.RepositoryContext.From(ctx)

	uc := uploadcache.Context.From(ctx)
	w, err := uc.Resume(ctx, req.ID)
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
