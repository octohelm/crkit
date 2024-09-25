package proxy

import (
	"context"
	"net/url"

	"github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/remote"
)

// namespace fetches content from a remote registry and caches it locally
type namespace struct {
	embedded content.Namespace // provides local registry functionality
	remote   content.Namespace
}

func NewProxyFallbackRegistry(ctx context.Context, registry content.Namespace, config remote.RegistryConfig) (content.Namespace, error) {
	remoteURL, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}

	r, err := remote.New(remoteURL.String(), remote.WithAuth(authn.FromConfig(authn.AuthConfig{
		Username: config.Username,
		Password: config.Password,
	})))
	if err != nil {
		return nil, err
	}

	return &namespace{
		embedded: registry,
		remote:   r,
	}, nil
}

func (n *namespace) Repository(ctx context.Context, name reference.Named) (content.Repository, error) {
	localRepo, err := n.embedded.Repository(ctx, name)
	if err != nil {
		return nil, err
	}

	remoteRepo, err := n.remote.Repository(ctx, name)
	if err != nil {
		return nil, err
	}

	return &repository{
		name:       name,
		localRepo:  localRepo,
		remoteRepo: remoteRepo,
	}, nil
}

type repository struct {
	name       reference.Named
	localRepo  content.Repository
	remoteRepo content.Repository
}

func (pr *repository) Named() reference.Named {
	return pr.name
}

func (pr *repository) Manifests(ctx context.Context) (content.ManifestService, error) {
	l, err := pr.localRepo.Manifests(ctx)
	if err != nil {
		return nil, err
	}
	r, err := pr.remoteRepo.Manifests(ctx)
	if err != nil {
		return nil, err
	}

	return &proxyManifestService{
		repositoryName:  pr.name,
		localManifests:  l,
		remoteManifests: r,
	}, nil
}

func (pr *repository) Blobs(ctx context.Context) (content.BlobStore, error) {
	l, err := pr.localRepo.Blobs(ctx)
	if err != nil {
		return nil, err
	}
	r, err := pr.remoteRepo.Blobs(ctx)
	if err != nil {
		return nil, err
	}
	return &proxyBlobStore{
		repositoryName: pr.name,
		localStore:     l,
		remoteStore:    r,
	}, nil
}

func (pr *repository) Tags(ctx context.Context) (content.TagService, error) {
	l, err := pr.localRepo.Tags(ctx)
	if err != nil {
		return nil, err
	}
	r, err := pr.remoteRepo.Tags(ctx)
	if err != nil {
		return nil, err
	}
	return &proxyTagService{
		localTags:  l,
		remoteTags: r,
	}, nil
}
