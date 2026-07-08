package remote

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/statuserror"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/apis/registry/v2"
	"github.com/octohelm/crkit/pkg/content"
	endpointsv2 "github.com/octohelm/crkit/pkg/endpoints/registry/v2"
)

type tagService struct {
	named  reference.Named
	client courier.Client
}

var _ content.TagService = &tagService{}

func (ts *tagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	req := &endpointsv2.HeadManifest{}
	req.Accept = strings.Join(slices.Collect(maps.Keys((&manifestv1.Payload{}).Mapping())), ",")
	req.Name = v2.Name(ts.named.Name())
	req.Reference = v2.Reference(tag)

	_, meta, err := Do(ctx, ts.client, req)
	if err != nil {
		errd := &statuserror.Descriptor{}
		if errors.As(err, &errd) {
			if errd.StatusCode() == 404 {
				return nil, &v2.ErrTagUnknown{
					Name: ts.named.Name(),
					Tag:  tag,
				}
			}
		}
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
	resolve := &endpointsv2.GetManifest{}
	resolve.Name = v2.Name(ts.named.Name())
	resolve.Reference = v2.Reference(desc.Digest.String())

	m, _, err := Do(ctx, ts.client, resolve)
	if err != nil {
		return err
	}

	put := &endpointsv2.PutManifest{}
	put.Name = v2.Name(ts.named.Name())
	put.Reference = v2.Reference(tag)
	put.Manifest = *m

	if _, _, err := Do(ctx, ts.client, put); err != nil {
		return err
	}
	return nil
}

func (ts *tagService) Untag(ctx context.Context, tag string) error {
	req := &endpointsv2.DeleteManifest{}
	req.Name = v2.Name(ts.named.Name())
	req.Reference = v2.Reference(tag)

	_, _, err := Do(ctx, ts.client, req)
	return err
}

func (ts *tagService) All(ctx context.Context) ([]string, error) {
	resolve := &endpointsv2.ListTag{}
	resolve.Name = v2.Name(ts.named.Name())

	list, _, err := Do(ctx, ts.client, resolve)
	if err != nil {
		return nil, err
	}

	return list.Tags, nil
}
