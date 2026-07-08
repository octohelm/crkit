package remote

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/statuserror"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	registryv2 "github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointsv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

type manifestService struct {
	named  reference.Named
	client courier.Client
}

var _ content.ManifestService = &manifestService{}

func (ms *manifestService) Delete(ctx context.Context, dgst digest.Digest) error {
	req := &endpointsv2.DeleteManifest{}
	req.Name = registryv2.Name(ms.named.Name())
	req.Reference = registryv2.Reference(dgst.String())

	_, _, err := Do(ctx, ms.client, req)
	return err
}

func (ms *manifestService) Put(ctx context.Context, m manifestv1.Manifest) (digest.Digest, error) {
	req := &endpointsv2.PutManifest{}
	req.Name = registryv2.Name(ms.named.Name())
	p, err := manifestv1.From(m)
	if err != nil {
		return "", err
	}

	_, dgst, err := p.Payload()
	if err != nil {
		return "", err
	}
	req.Manifest = *p
	req.Reference = registryv2.Reference(dgst.String())

	_, meta, err := Do(ctx, ms.client, req)
	if err != nil {
		return "", err
	}

	returns := digest.Digest(meta.Get("Docker-Content-Digest"))
	if returns != dgst {
		return "", fmt.Errorf("expect %s but got %s: %w", dgst, returns, &registryv2.ErrManifestUnverified{})
	}

	return returns, nil
}

func (ms *manifestService) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	req := &endpointsv2.HeadManifest{}
	req.Name = registryv2.Name(ms.named.Name())
	req.Accept = strings.Join(slices.Collect(maps.Keys((&manifestv1.Payload{}).Mapping())), ",")
	req.Reference = registryv2.Reference(dgst.String())

	_, meta, err := Do(ctx, ms.client, req)
	if err != nil {
		errd := &statuserror.Descriptor{}
		if errors.As(err, &errd) {
			if errd.StatusCode() == 404 {
				return nil, &registryv2.ErrManifestUnknownRevision{
					Name:     ms.named.Name(),
					Revision: dgst,
				}
			}
		}
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
	req := &endpointsv2.GetManifest{}
	req.Accept = strings.Join(slices.Collect(maps.Keys((&manifestv1.Payload{}).Mapping())), ",")
	req.Name = registryv2.Name(ms.named.Name())
	req.Reference = registryv2.Reference(dgst.String())

	p, _, err := Do(ctx, ms.client, req)
	if err != nil {
		return nil, err
	}

	return p, nil
}
