package remote

import (
	"context"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
	"golang.org/x/exp/maps"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

type manifestService struct {
	named  reference.Named
	client *Client
}

var _ content.ManifestService = &manifestService{}

func (ms *manifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	req := &registry.DeleteManifest{}
	req.Name = content.Name(ms.named.Name())
	req.Reference = content.Reference(dgst.String())

	_, _, err := Do(ctx, ms.client, req)
	return err
}

func (ms *manifestService) Put(ctx context.Context, m manifestv1.Manifest) (digest.Digest, error) {
	req := &registry.PutManifest{}
	req.Name = content.Name(ms.named.Name())
	p, err := manifestv1.From(m)
	if err != nil {
		return "", err
	}

	_, dgst, err := p.Payload()
	if err != nil {
		return "", err
	}
	req.Manifest = *p
	req.Reference = content.Reference(dgst.String())

	_, meta, err := Do(ctx, ms.client, req)
	if err != nil {
		return "", err
	}
	return digest.Digest(meta.Get("Docker-Content-Digest")), nil
}

func (ms *manifestService) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	req := &registry.HeadManifest{}
	req.Name = content.Name(ms.named.Name())
	req.Accept = strings.Join(maps.Keys((&manifestv1.Payload{}).Mapping()), ",")
	req.Reference = content.Reference(dgst.String())

	_, meta, err := Do(ctx, ms.client, req)
	if err != nil {
		return nil, err
	}

	i, _ := strconv.ParseInt(meta.Get("Content-Length"), 10, 64)

	return &manifestv1.Descriptor{
		MediaType: meta.Get("Content-Type"),
		Digest:    digest.Digest(meta.Get("Docker-Content-Digest")),
		Size:      i,
	}, nil
}

func (ms *manifestService) Get(ctx context.Context, dgst digest.Digest) (manifestv1.Manifest, error) {
	req := &registry.GetManifest{}
	req.Accept = strings.Join(maps.Keys((&manifestv1.Payload{}).Mapping()), ",")
	req.Name = content.Name(ms.named.Name())
	req.Reference = content.Reference(dgst.String())

	p, _, err := Do(ctx, ms.client, req)
	if err != nil {
		return nil, err
	}
	return p, nil
}
