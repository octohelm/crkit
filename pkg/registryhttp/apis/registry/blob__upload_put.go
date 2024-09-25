package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	registryoperator "github.com/octohelm/crkit/pkg/registryhttp/apis/registry/operator"
	"github.com/octohelm/crkit/pkg/uploadcache"
	"github.com/opencontainers/go-digest"
)

func (UploadPutBlob) MiddleOperators() courier.MiddleOperators {
	return courier.MiddleOperators{
		&registryoperator.NameScoped{},
	}
}

type UploadPutBlob struct {
	courierhttp.MethodPut `path:"/blobs/uploads/{id}"`

	ID            string         `name:"id" in:"path"`
	ContentLength int            `name:"Content-Length,omitempty" in:"header"`
	Digest        content.Digest `name:"digest" in:"query"`
	Chunk         io.ReadCloser  `in:"body"`
}

func (req *UploadPutBlob) Output(ctx context.Context) (any, error) {
	defer req.Chunk.Close()

	repo := content.RepositoryContext.From(ctx)

	uc := uploadcache.Context.From(ctx)
	w, err := uc.Resume(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if req.ContentLength > 0 {
		if _, err := io.Copy(w, req.Chunk); err != nil {
			return nil, err
		}
	}

	d, err := w.Commit(ctx, manifestv1.Descriptor{
		Digest: digest.Digest(req.Digest),
	})
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(http.StatusCreated),
		courierhttp.WithMetadata("Docker-Content-Digest", d.Digest.String()),
		courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/%s", repo.Named().Name(), d.Digest.String())),
	), nil
}
