package remote

import (
	"context"
	"net/http"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/pkg/errors"

	"github.com/opencontainers/go-digest"

	"github.com/distribution/distribution/v3"
)

type tagService struct {
	*repository
}

var _ distribution.TagService = &tagService{}

func (b *tagService) normalizeError(tag string, err error) error {
	terr := &transport.Error{}
	if errors.As(err, &terr) {
		if terr.StatusCode == http.StatusNotFound {
			return distribution.ErrManifestUnknown{
				Name: b.named.Name(),
				Tag:  tag,
			}
		}
	}
	return nil
}

func (pt *tagService) Get(ctx context.Context, tag string) (distribution.Descriptor, error) {
	d, err := pt.puller.Get(ctx, pt.repo.Tag(tag))
	if err != nil {
		return distribution.Descriptor{}, pt.normalizeError(tag, err)
	}
	return distribution.Descriptor{
		MediaType:   string(d.MediaType),
		Digest:      digest.NewDigestFromHex(d.Digest.Algorithm, d.Digest.Hex),
		Size:        d.Size,
		Annotations: d.Annotations,
	}, nil
}

func (pt *tagService) Tag(ctx context.Context, tag string, desc distribution.Descriptor) error {
	d, err := pt.puller.Get(ctx, pt.repo.Digest(desc.Digest.String()))
	if err != nil {
		return err
	}
	return pt.pusher.Push(ctx, pt.repo.Tag(tag), d)
}

func (pt *tagService) Untag(ctx context.Context, tag string) error {
	return pt.pusher.Delete(ctx, pt.repo.Tag(tag))
}

func (pt *tagService) All(ctx context.Context) ([]string, error) {
	return pt.puller.List(ctx, pt.repo)
}

func (pt *tagService) Lookup(ctx context.Context, digest distribution.Descriptor) ([]string, error) {
	return nil, nil
}
