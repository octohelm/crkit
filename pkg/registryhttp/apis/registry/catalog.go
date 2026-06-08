package registry

import (
	"context"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/collect"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type Catalog struct {
	endpointregistryv2.Catalog

	namespace content.Namespace `inject:""`
}

func (r *Catalog) Output(ctx context.Context) (any, error) {
	names, err := collect.Catalogs(ctx, r.namespace)
	if err != nil {
		return nil, err
	}
	return &apiregistryv2.CatalogResponse{Repositories: names}, nil
}
