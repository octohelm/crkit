package registry

import (
	"context"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
)

type HeadBlob struct {
	courierhttp.MethodHead `path:"/{name...}/blobs/{digest}"`

	NameScoped

	Digest content.Digest `name:"digest" in:"path"`
}

func (req *HeadBlob) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
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

	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#checking-if-content-exists-in-the-registry
	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(200),
		courierhttp.WithMetadata("Docker-Content-Digest", desc.Digest.String()),
		courierhttp.WithMetadata("Content-Type", desc.MediaType),
		courierhttp.WithMetadata("Content-Length", fmt.Sprintf("%d", desc.Size)),
	), nil
}
