package remote

import (
	"context"
	"fmt"
	"io"
	"iter"
	"sync"

	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	syncx "github.com/octohelm/x/sync"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func pullAsImage(ctx context.Context, repo content.Repository, desc ocispecv1.Descriptor, open internal.Opener) (oci.Image, error) {
	img := &image{
		repo: repo,
	}

	r, err := open(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := img.InitFromReader(r, desc); err != nil {
		return nil, fmt.Errorf("init image %s failed: %w", desc.Digest, err)
	}

	return img, nil
}

type image struct {
	internal.Image

	repo content.Repository

	cached syncx.Map[digest.Digest, func() oci.Blob]
}

func (i *image) fetch(ctx context.Context, d ocispecv1.Descriptor) oci.Blob {
	get, _ := i.cached.LoadOrStore(d.Digest, sync.OnceValue(func() oci.Blob {
		return partial.BlobFromOpener(func(ctx context.Context) (io.ReadCloser, error) {
			blobs, err := i.repo.Blobs(ctx)
			if err != nil {
				return nil, err
			}
			return blobs.Open(ctx, d.Digest)
		}, d)
	}))
	return get()
}

func (i *image) Config(ctx context.Context) (oci.Blob, error) {
	m, err := i.Value(ctx)
	if err != nil {
		return nil, err
	}

	return i.fetch(ctx, m.Config), nil
}

func (i *image) Layers(ctx context.Context) iter.Seq2[oci.Blob, error] {
	return func(yield func(oci.Blob, error) bool) {
		img, err := i.Value(ctx)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, l := range img.Layers {
			if !yield(i.fetch(ctx, l), nil) {
				return
			}
		}
	}
}
