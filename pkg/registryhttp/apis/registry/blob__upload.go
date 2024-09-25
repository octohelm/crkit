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

func (UploadBlob) MiddleOperators() courier.MiddleOperators {
	return courier.MiddleOperators{
		&registryoperator.NameScoped{},
	}
}

type UploadBlob struct {
	courierhttp.MethodPost `path:"/blobs/uploads"`

	ContentLength int            `name:"Content-Length,omitempty" in:"header"`
	ContentType   string         `name:"Content-Type,omitempty" in:"header"`
	Digest        content.Digest `name:"digest,omitempty" in:"query"`
	Blob          io.ReadCloser  `in:"body"`
}

func (req *UploadBlob) Output(ctx context.Context) (any, error) {
	defer req.Blob.Close()

	repo := content.RepositoryContext.From(ctx)

	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#single-post
	if req.Digest != "" {
		blobs, err := repo.Blobs(ctx)
		if err != nil {
			return nil, err
		}
		w, err := blobs.Writer(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, req.Blob); err != nil {
			return nil, err
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

	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#post-then-put
	uc := uploadcache.Context.From(ctx)

	w, err := uc.BlobWriter(ctx, repo)
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(http.StatusAccepted),
		courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo.Named().Name(), w.ID())),
	), nil
}
