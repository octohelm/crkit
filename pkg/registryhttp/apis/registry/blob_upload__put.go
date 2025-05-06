package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/octohelm/courier/pkg/courierhttp"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	contentutil "github.com/octohelm/crkit/pkg/content/util"
	"github.com/opencontainers/go-digest"
)

// +gengo:injectable
type PutBlobUpload struct {
	courierhttp.MethodPut `path:"/{name...}/blobs/uploads/{id}"`

	NameScoped

	ID string `name:"id" in:"path"`

	ContentRange  contentutil.Range `name:"Content-Range,omitzero" in:"header"`
	ContentLength int64             `name:"Content-Length,omitempty" in:"header"`

	Digest content.Digest `name:"digest" in:"query"`
	Chunk  io.ReadCloser  `in:"body"`
}

func (req *PutBlobUpload) Output(ctx context.Context) (any, error) {
	defer req.Chunk.Close()

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

	if req.ContentLength > 0 || !req.ContentRange.IsZero() {
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
