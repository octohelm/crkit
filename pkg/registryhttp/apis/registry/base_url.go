package registry

import (
	"context"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

type BaseURL struct {
	endpointregistryv2.BaseURL
}

func (r *BaseURL) Output(ctx context.Context) (any, error) {
	return map[string]string{}, nil
}

func repository(ctx context.Context, ns content.Namespace, name apiregistryv2.Name) (content.Repository, error) {
	return ns.Repository(ctx, name)
}
