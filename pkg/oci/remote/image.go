package remote

import (
	"context"
	"fmt"
	"io"
	"iter"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func pullAsImage(ctx context.Context, repo content.Repository, desc ocispecv1.Descriptor, opener internal.Opener) (oci.Image, error) {
	img := &image{
		repo: repo,
	}

	raw, err := internal.ReadAllFromOpener(ctx, opener)
	if err != nil {
		return nil, err
	}

	if err := img.InitFromRaw(raw, desc); err != nil {
		return nil, fmt.Errorf("init image %s failed: %w", desc.Digest, err)
	}

	return img, nil
}

type image struct {
	internal.Image

	repo content.Repository
}

func (i *image) Config(ctx context.Context) (oci.Blob, error) {
	m, err := i.Value(ctx)
	if err != nil {
		return nil, err
	}

	return partial.BlobFromOpener(func(ctx context.Context) (io.ReadCloser, error) {
		blobs, err := i.repo.Blobs(ctx)
		if err != nil {
			return nil, err
		}
		return blobs.Open(ctx, m.Config.Digest)
	}, m.Config), nil
}

func (i *image) Layers(ctx context.Context) iter.Seq2[oci.Blob, error] {
	return func(yield func(oci.Blob, error) bool) {
		img, err := i.Value(ctx)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, l := range img.Layers {
			layer := partial.BlobFromOpener(func(ctx context.Context) (io.ReadCloser, error) {
				blobs, err := i.repo.Blobs(ctx)
				if err != nil {
					return nil, err
				}
				return blobs.Open(ctx, l.Digest)
			}, l)

			if !yield(layer, nil) {
				return
			}
		}
	}
}
