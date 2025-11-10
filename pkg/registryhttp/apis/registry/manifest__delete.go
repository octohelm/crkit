package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"

	"github.com/octohelm/crkit/pkg/content"
)

type DeleteManifest struct {
	courierhttp.MethodDelete `path:"/{name...}/manifests/{reference}"`

	NameScoped

	Reference content.Reference `name:"reference" in:"path"`
}

func (req *DeleteManifest) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
	if err != nil {
		return nil, err
	}

	dgst, err := req.Reference.Digest()
	if err == nil {
		manifests, err := repo.Manifests(ctx)
		if err != nil {
			return nil, err
		}

		if err := manifests.Delete(ctx, dgst); err != nil {
			return nil, err
		}

		return nil, nil
	}

	tag := string(req.Reference)
	tags, err := repo.Tags(ctx)
	if err != nil {
		return nil, err
	}
	if err := tags.Untag(ctx, tag); err != nil {
		return nil, err
	}
	return nil, nil
}
