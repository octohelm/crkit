package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
)

type GetManifest struct {
	courierhttp.MethodGet `path:"/{name...}/manifests/{reference}"`

	NameScoped
	Accept    string            `name:"Accept,omitempty" in:"header"`
	Reference content.Reference `name:"reference" in:"path"`
}

func (req *GetManifest) Output(ctx context.Context) (any, error) {
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

	m, err := manifests.Get(ctx, dgst)
	if err != nil {
		return nil, err
	}

	p, err := manifestv1.From(m)
	if err != nil {
		return nil, err
	}

	return courierhttp.Wrap(p,
		courierhttp.WithMetadata("Docker-Content-Digest", dgst.String()),
		courierhttp.WithMetadata("Content-Type", m.Type()),
	), nil
}
