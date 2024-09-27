package remote

import (
	"context"
	"strconv"

	"github.com/opencontainers/go-digest"

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
)

type tagService struct {
	named  reference.Named
	client *Client
}

var _ content.TagService = &tagService{}

func (ts *tagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	req := &registry.HeadManifest{}
	req.Name = content.Name(ts.named.Name())
	req.Reference = content.Reference(tag)

	_, meta, err := Do(ctx, ts.client, req)
	if err != nil {
		return nil, err
	}

	i, _ := strconv.ParseInt(meta.Get("Content-Length"), 64, 10)

	return &manifestv1.Descriptor{
		MediaType: meta.Get("Content-Type"),
		Digest:    digest.Digest(meta.Get("Docker-Content-Digest")),
		Size:      i,
	}, nil
}

func (ts *tagService) Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error {
	resolve := &registry.GetManifest{}
	resolve.Name = content.Name(ts.named.Name())
	resolve.Reference = content.Reference(desc.Digest.String())

	m, _, err := Do(ctx, ts.client, resolve)
	if err != nil {
		return err
	}

	put := &registry.PutManifest{}
	put.Name = content.Name(ts.named.Name())
	put.Reference = content.Reference(tag)
	put.Manifest = *m

	if _, _, err := Do(ctx, ts.client, put); err != nil {
		return err
	}
	return nil
}

func (ts *tagService) Untag(ctx context.Context, tag string) error {
	req := &registry.DeleteManifest{}
	req.Name = content.Name(ts.named.Name())
	req.Reference = content.Reference(tag)

	_, _, err := Do(ctx, ts.client, req)
	return err
}

func (ts *tagService) All(ctx context.Context) ([]string, error) {
	resolve := &registry.ListTag{}
	resolve.Name = content.Name(ts.named.Name())

	list, _, err := Do(ctx, ts.client, resolve)
	if err != nil {
		return nil, err
	}

	return list.Tags, nil
}
