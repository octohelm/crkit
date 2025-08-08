package proxy

import (
	"context"
	"fmt"

	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

type proxyManifestService struct {
	repositoryName  reference.Named
	localManifests  content.ManifestService
	remoteManifests content.ManifestService
}

var _ content.ManifestService = &proxyManifestService{}

func (pms *proxyManifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	return pms.localManifests.Delete(ctx, dgst)
}

func (pms *proxyManifestService) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	info, err := pms.localManifests.Info(ctx, dgst)
	if err == nil {
		return info, nil
	}
	return pms.remoteManifests.Info(ctx, dgst)
}

func (pms *proxyManifestService) Get(ctx context.Context, dgst digest.Digest) (manifestv1.Manifest, error) {
	manifest, err := pms.localManifests.Get(ctx, dgst)
	if err != nil {
		manifest, err = pms.remoteManifests.Get(ctx, dgst)
		if err != nil {
			return nil, err
		}

		go func() {
			if _, err := pms.localManifests.Put(context.WithoutCancel(ctx), manifest); err != nil {
				logr.FromContext(ctx).Error(fmt.Errorf("store manifest to local failed: %w", err))
			}
		}()
	}
	return manifest, err
}

func (pms *proxyManifestService) Put(ctx context.Context, manifest manifestv1.Manifest) (digest.Digest, error) {
	return pms.localManifests.Put(ctx, manifest)
}
