package registry

import (
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/courierhttp"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type GetBlob struct {
	endpointregistryv2.GetBlob

	namespace content.Namespace `inject:""`
}

func (req *GetBlob) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}

	desc, err := blobs.Info(ctx, digest.Digest(req.Digest))
	if err != nil {
		return nil, err
	}

	b, err := blobs.Open(ctx, digest.Digest(req.Digest))
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap(
		b,
		courierhttp.WithMetadata("Docker-Content-Digest", string(req.Digest)),
		courierhttp.WithMetadata("Content-Type", desc.MediaType),
		courierhttp.WithMetadata("Content-Length", fmt.Sprintf("%d", desc.Size)),
	), nil
}
