package proxy

import (
	"context"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/octohelm/crkit/pkg/registry/remote"
	"net/url"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/reference"
)

// namespace fetches content from a remote registry and caches it locally
type namespace struct {
	embedded distribution.Namespace // provides local registry functionality
	remote   distribution.Namespace
}

func NewProxyFallbackRegistry(ctx context.Context, registry distribution.Namespace, driver driver.StorageDriver, config configuration.Proxy) (distribution.Namespace, error) {
	remoteURL, err := url.Parse(config.RemoteURL)
	if err != nil {
		return nil, err
	}

	r, err := remote.New(remoteURL.String(), authn.FromConfig(authn.AuthConfig{
		Username: config.Username,
		Password: config.Password,
	}))
	if err != nil {
		return nil, err
	}

	return &namespace{
		embedded: registry,
		remote:   r,
	}, nil
}

func (n *namespace) Scope() distribution.Scope {
	return distribution.GlobalScope
}

func (n *namespace) Repositories(ctx context.Context, repos []string, last string) (int, error) {
	return n.embedded.Repositories(ctx, repos, last)
}

func (n *namespace) Repository(ctx context.Context, name reference.Named) (distribution.Repository, error) {
	localRepo, err := n.embedded.Repository(ctx, name)
	if err != nil {
		return nil, err
	}

	localManifests, err := localRepo.Manifests(ctx, storage.SkipLayerVerification())
	if err != nil {
		return nil, err
	}

	remoteRepo, err := n.remote.Repository(ctx, name)
	if err != nil {
		return nil, err
	}

	remoteManifests, err := remoteRepo.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	return &repository{
		blobStore: &proxyBlobStore{
			repositoryName: name,
			localStore:     localRepo.Blobs(ctx),
			remoteStore:    remoteRepo.Blobs(ctx),
		},
		manifests: &proxyManifestService{
			repositoryName:  name,
			remoteManifests: remoteManifests,
			localManifests:  localManifests,
		},
		name: name,
		tags: &proxyTagService{
			remoteTags: remoteRepo.Tags(ctx),
			localTags:  localRepo.Tags(ctx),
		},
	}, nil
}

func (n *namespace) Blobs() distribution.BlobEnumerator {
	return n.embedded.Blobs()
}

func (n *namespace) BlobStatter() distribution.BlobStatter {
	return n.embedded.BlobStatter()
}

type repository struct {
	blobStore distribution.BlobStore
	manifests distribution.ManifestService
	name      reference.Named
	tags      distribution.TagService
}

func (pr *repository) Manifests(ctx context.Context, options ...distribution.ManifestServiceOption) (distribution.ManifestService, error) {
	return pr.manifests, nil
}

func (pr *repository) Blobs(ctx context.Context) distribution.BlobStore {
	return pr.blobStore
}

func (pr *repository) Named() reference.Named {
	return pr.name
}

func (pr *repository) Tags(ctx context.Context) distribution.TagService {
	return pr.tags
}
