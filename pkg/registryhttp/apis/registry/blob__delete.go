package registry

import (
	"context"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/courierhttp"

	"github.com/octohelm/crkit/pkg/content"
)

type DeleteBlob struct {
	courierhttp.MethodDelete `path:"/{name...}/blobs/{digest}"`

	NameScoped

	Digest content.Digest `name:"digest" in:"path"`
}

func (req *DeleteBlob) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
	if err != nil {
		return nil, err
	}

	blobs, err := repo.Blobs(ctx)
	if err != nil {
		return nil, err
	}

	err = blobs.Remove(ctx, digest.Digest(req.Digest))
	if err != nil {
		return nil, err
	}
	return nil, nil
}
