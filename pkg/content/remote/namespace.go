package remote

import (
	"context"
	"net/url"
	"strings"

	"github.com/distribution/reference"
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/crkit/pkg/content"
)

func WithClient(c courier.Client) Option {
	return func(n *namespace) {
		n.client = c
	}
}

type Option func(n *namespace)

func New(ctx context.Context, registry Registry, options ...Option) (content.Namespace, error) {

	remoteURI, err := url.Parse(registry.Endpoint)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Registry: registry,
	}

	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	n := &namespace{
		client: c,
	}

	for _, opt := range options {
		opt(n)
	}

	n.remoteURI = remoteURI

	return n, nil
}

type namespace struct {
	client    courier.Client
	remoteURI *url.URL
}

func (n *namespace) Repository(ctx context.Context, named reference.Named) (content.Repository, error) {
	name := named.Name()

	if strings.HasPrefix(name, n.remoteURI.Host) {
		named = content.Name(name[len(n.remoteURI.Host)+1:])
	}

	return &repository{
		named:  named,
		client: n.client,
	}, nil
}

type repository struct {
	client courier.Client
	named  reference.Named
}

func (r *repository) Named() reference.Named {
	return r.named
}

func (r *repository) Manifests(ctx context.Context) (content.ManifestService, error) {
	return &manifestService{
		named:  r.named,
		client: r.client,
	}, nil
}

func (r *repository) Blobs(ctx context.Context) (content.BlobStore, error) {
	return &blobStore{
		named:  r.named,
		client: r.client,
	}, nil
}

func (r *repository) Tags(ctx context.Context) (content.TagService, error) {
	return &tagService{
		named:  r.named,
		client: r.client,
	}, nil
}
