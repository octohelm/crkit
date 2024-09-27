package proxy

import (
	"context"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
)

type proxyTagService struct {
	localTags  content.TagService
	remoteTags content.TagService
}

var _ content.TagService = &proxyTagService{}

func (pt *proxyTagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	desc, err := pt.remoteTags.Get(ctx, tag)
	if err == nil {
		return desc, nil
	}

	desc, err = pt.localTags.Get(ctx, tag)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (pt proxyTagService) Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error {
	return pt.localTags.Tag(ctx, tag, desc)
}

func (pt proxyTagService) Untag(ctx context.Context, tag string) error {
	err := pt.localTags.Untag(ctx, tag)
	if err != nil {
		return err
	}
	return nil
}

func (pt proxyTagService) All(ctx context.Context) ([]string, error) {
	tags, err := pt.remoteTags.All(ctx)
	if err == nil {
		return tags, err
	}
	return pt.localTags.All(ctx)
}
