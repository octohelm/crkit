package content

import (
	"context"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
)

type TagService interface {
	Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error)
	Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error
	Untag(ctx context.Context, tag string) error
	All(ctx context.Context) ([]string, error)
}

type TagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (TagList) ContentType() string {
	return "application/json"
}
