package registry

import (
	"context"

	apiregistryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointregistryv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

// +gengo:injectable
type ListTag struct {
	endpointregistryv2.ListTag

	namespace content.Namespace `inject:""`
}

func (req *ListTag) Output(ctx context.Context) (any, error) {
	repo, err := repository(ctx, req.namespace, apiregistryv2.Name(req.Name))
	if err != nil {
		return nil, err
	}

	tags, err := repo.Tags(ctx)
	if err != nil {
		return nil, err
	}

	tagList, err := tags.All(ctx)
	if err != nil {
		return nil, err
	}

	return &apiregistryv2.TagList{
		Name: req.Name.Name(),
		Tags: tagList,
	}, nil
}
