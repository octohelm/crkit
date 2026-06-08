package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/courierhttp"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// CreateBlobUpload
// +gengo:injectable
type CreateBlobUpload struct {
	endpointregistryv2.CreateBlobUpload

	namespace content.Namespace `inject:""`
}

func (req *CreateBlobUpload) Output(ctx context.Context) (any, error) {
	defer req.Blob.Close()

	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

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
		defer w.Close()

		if _, err := io.Copy(w, req.Blob); err != nil {
			return nil, err
		}

		d, err := w.Commit(ctx, manifestv1.Descriptor{
			Digest: digest.Digest(req.Digest),
		})
		if err != nil {
			return nil, err
		}

		return courierhttp.Wrap[any](
			nil,
			courierhttp.WithStatusCode(http.StatusCreated),
			courierhttp.WithMetadata("Docker-Content-Digest", d.Digest.String()),
			courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/%s", repo.Named().Name(), d.Digest.String())),
		), nil
	}

	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}

	w, err := blobs.Writer(ctx)
	if err != nil {
		return nil, err
	}
	defer w.Close()

	return courierhttp.Wrap[any](
		nil,
		courierhttp.WithStatusCode(http.StatusAccepted),
		courierhttp.WithMetadata("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo.Named().Name(), w.ID())),
	), nil
}
