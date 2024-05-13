package proxy

import (
	"context"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type proxyManifestService struct {
	repositoryName  reference.Named
	localManifests  distribution.ManifestService
	remoteManifests distribution.ManifestService
}

var _ distribution.ManifestService = &proxyManifestService{}

func (pms proxyManifestService) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	exists, err := pms.localManifests.Exists(ctx, dgst)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	return pms.remoteManifests.Exists(ctx, dgst)
}

func (pms proxyManifestService) Get(ctx context.Context, dgst digest.Digest, options ...distribution.ManifestServiceOption) (distribution.Manifest, error) {
	manifest, err := pms.localManifests.Get(ctx, dgst, options...)
	if err != nil {
		manifest, err = pms.remoteManifests.Get(ctx, dgst, options...)
		if err != nil {
			return nil, err
		}
		// store local
		go func() {
			if _, err := pms.localManifests.Put(ctx, manifest, storage.SkipLayerVerification()); err != nil {
				logr.FromContext(ctx).Error(errors.Wrapf(err, "store manifest to local failed"))
			}
		}()
	}
	return manifest, err
}

func (pms proxyManifestService) Put(ctx context.Context, manifest distribution.Manifest, options ...distribution.ManifestServiceOption) (digest.Digest, error) {
	return pms.localManifests.Put(ctx, manifest, options...)
}

func (pms proxyManifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	return pms.localManifests.Delete(ctx, dgst)
}
