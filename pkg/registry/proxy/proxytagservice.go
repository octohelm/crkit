package proxy

import (
	"context"

	"github.com/distribution/distribution/v3"
)

type proxyTagService struct {
	localTags  distribution.TagService
	remoteTags distribution.TagService
}

var _ distribution.TagService = proxyTagService{}

func (pt proxyTagService) Get(ctx context.Context, tag string) (distribution.Descriptor, error) {
	desc, err := pt.remoteTags.Get(ctx, tag)
	if err == nil {
		err := pt.localTags.Tag(ctx, tag, desc)
		if err != nil {
			return distribution.Descriptor{}, err
		}
		return desc, nil
	}

	desc, err = pt.localTags.Get(ctx, tag)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	return desc, nil
}

func (pt proxyTagService) Tag(ctx context.Context, tag string, desc distribution.Descriptor) error {
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

func (pt proxyTagService) Lookup(ctx context.Context, digest distribution.Descriptor) ([]string, error) {
	return pt.localTags.Lookup(ctx, digest)
}
