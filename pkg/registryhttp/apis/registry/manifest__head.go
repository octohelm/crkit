package registry

import (
	"context"
	"fmt"

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
)

type HeadManifest struct {
	courierhttp.MethodHead `path:"/{name...}/manifests/{reference}"`

	NameScoped

	Accept    string            `name:"Accept,omitempty" in:"header"`
	Reference content.Reference `name:"reference" in:"path"`
}

func (req *HeadManifest) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
	if err != nil {
		return nil, err
	}

	dgst, err := req.Reference.Digest()
	if err != nil {
		tags, err := repo.Tags(ctx)
		if err != nil {
			return nil, err
		}

		d, err := tags.Get(ctx, string(req.Reference))
		if err != nil {
			return nil, err
		}
		dgst = d.Digest
	}

	manifests, err := repo.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	desc, err := manifests.Info(ctx, dgst)
	if err != nil {
		return nil, err
	}

	m, err := manifests.Get(ctx, dgst)
	if err != nil {
		return nil, err
	}

	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#checking-if-content-exists-in-the-registry
	return courierhttp.Wrap[any](nil,
		courierhttp.WithStatusCode(200),
		courierhttp.WithMetadata("Docker-Content-Digest", desc.Digest.String()),
		courierhttp.WithMetadata("Content-Length", fmt.Sprintf("%d", desc.Size)),
		courierhttp.WithMetadata("Content-Type", m.Type()),
	), nil
}
