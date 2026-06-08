package content

import (
	"context"
	"iter"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
)

type TagService interface {
	Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error)
	Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error
	Untag(ctx context.Context, tag string) error
	All(ctx context.Context) ([]string, error)
}

type TagRevisionIterable interface {
	TagRevisions(ctx context.Context, tag string) iter.Seq2[LinkedDigest, error]
}

// Deprecated: use apiregistryv2.TagList instead.
//
//go:fix inline
type TagList = registryv2.TagList
