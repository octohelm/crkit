package registry

import (
	"context"

	"github.com/octohelm/courier/pkg/courierhttp"
	"github.com/octohelm/crkit/pkg/content"
)

type ListTag struct {
	courierhttp.MethodGet `path:"/{name...}/tags/list"`

	NameScoped
}

func (req *ListTag) Output(ctx context.Context) (any, error) {
	repo, err := req.Repository(ctx)
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

	return &content.TagList{
		Name: req.Name.Name(),
		Tags: tagList,
	}, nil
}
