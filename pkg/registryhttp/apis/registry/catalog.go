package registry

import (
	"context"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/collect"

	"github.com/octohelm/courier/pkg/courierhttp"
)

// +gengo:injectable
type Catalog struct {
	courierhttp.MethodGet `path:"/_catalog" method:"get"`

	namespace content.Namespace `inject:""`
}

func (r *Catalog) Output(ctx context.Context) (any, error) {
	names, err := collect.Catalogs(ctx, r.namespace)
	if err != nil {
		return nil, err
	}
	return map[string][]string{"repositories": names}, nil
}
