package remote

import (
	"context"
	"net/http"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type tagService struct {
	*repository
}

var _ content.TagService = &tagService{}

func (b *tagService) normalizeError(tag string, err error) error {
	terr := &transport.Error{}
	if errors.As(err, &terr) {
		if terr.StatusCode == http.StatusNotFound {
			return &content.ErrTagUnknown{
				Tag: tag,
			}
		}
	}
	return err
}

func (pt *tagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	d, err := pt.puller.Get(ctx, pt.repo.Tag(tag))
	if err != nil {
		return nil, pt.normalizeError(tag, err)
	}
	return &manifestv1.Descriptor{
		MediaType:   string(d.MediaType),
		Digest:      digest.NewDigestFromHex(d.Digest.Algorithm, d.Digest.Hex),
		Size:        d.Size,
		Annotations: d.Annotations,
	}, nil
}

func (pt *tagService) Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error {
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
