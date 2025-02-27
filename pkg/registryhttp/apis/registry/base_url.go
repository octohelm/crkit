package registry

import (
	"context"

	"github.com/octohelm/crkit/pkg/content"

	"github.com/octohelm/courier/pkg/courierhttp"
)

type BaseURL struct {
	courierhttp.MethodGet
}

func (r *BaseURL) Output(ctx context.Context) (any, error) {
	return map[string]string{}, nil
}

// +gengo:injectable
type NameScoped struct {
	Name content.Name `name:"name" in:"path"`

	namespace content.Namespace `inject:""`
}

func (req *NameScoped) Repository(ctx context.Context) (content.Repository, error) {
	return req.namespace.Repository(ctx, req.Name)
}
