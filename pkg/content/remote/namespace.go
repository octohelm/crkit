package remote

import (
	"context"
	"net/url"
	"strings"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/content"
)

func New(ctx context.Context, registry Registry) (content.Namespace, error) {
	n := &namespace{
		client: &Client{
			Registry: registry,
		},
	}

	remoteURI, err := url.Parse(registry.Endpoint)
	if err != nil {
		return nil, err
	}

	n.remoteURI = remoteURI

	if err := n.client.Init(ctx); err != nil {
		return nil, err
	}

	return n, nil
}

type namespace struct {
	client    *Client
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
	client *Client
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
