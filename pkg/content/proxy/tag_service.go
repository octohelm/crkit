package proxy

import (
	"context"
	"fmt"

	"github.com/octohelm/x/logr"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
)

type proxyTagService struct {
	localTagService      content.TagService
	localManifestService content.ManifestService

	remoteTagService      content.TagService
	remoteManifestService content.ManifestService
}

var _ content.TagService = &proxyTagService{}

func (pt *proxyTagService) syncToLocalManifest(ctx context.Context, tag string, d *manifestv1.Descriptor) error {
	m, err := pt.remoteManifestService.Get(ctx, d.Digest)
	if err != nil {
		return err
	}
	dgst, err := pt.localManifestService.Put(ctx, m)
	if err != nil {
		return err
	}
	if err := pt.localTagService.Tag(ctx, tag, manifestv1.Descriptor{Digest: dgst}); err != nil {
		return err
	}
	return nil
}

func (pt *proxyTagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	remote, err := pt.remoteTagService.Get(ctx, tag)
	if err == nil {
		go func() {
			if err := pt.syncToLocalManifest(context.WithoutCancel(ctx), tag, remote); err != nil {
				logr.FromContext(ctx).Error(fmt.Errorf("store tagged manifest to local failed: %w", err))
			}
		}()
		return remote, nil
	}
	local, err := pt.localTagService.Get(ctx, tag)
	if err != nil {
		return nil, err
	}
	return local, nil
}

func (pt *proxyTagService) Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error {
	return pt.localTagService.Tag(ctx, tag, desc)
}

func (pt *proxyTagService) Untag(ctx context.Context, tag string) error {
	return pt.localTagService.Untag(ctx, tag)
}

func (pt *proxyTagService) All(ctx context.Context) ([]string, error) {
	return pt.localTagService.All(ctx)
}
