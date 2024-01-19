package distribution

import "context"

type TagService interface {
	All(ctx context.Context) ([]string, error)
	Get(ctx context.Context, tag string) (Descriptor, error)
	Tag(ctx context.Context, tag string, desc Descriptor) error
	Untag(ctx context.Context, tag string) error
}
