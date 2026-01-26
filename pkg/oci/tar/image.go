package tar

import (
	"context"
	"io"
	"iter"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func openAsImage(ctx context.Context, fileOpener FileOpener, desc ocispecv1.Descriptor, opener Opener) (oci.Image, error) {
	img := &image{
		fileOpener: fileOpener,
	}

	raw, err := internal.ReadAllFromOpener(ctx, opener)
	if err != nil {
		return nil, err
	}

	if err := img.InitFromRaw(raw, desc); err != nil {
		return nil, err
	}

	return img, nil
}

type image struct {
	internal.Image

	fileOpener FileOpener
}

func (i *image) Config(ctx context.Context) (oci.Blob, error) {
	m, err := i.Value(ctx)
	if err != nil {
		return nil, err
	}

	return partial.BlobFromOpener(func(ctx context.Context) (io.ReadCloser, error) {
		return i.fileOpener.Open(LayoutBlobsPath(m.Config.Digest))
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
				return i.fileOpener.Open(LayoutBlobsPath(l.Digest))
			}, l)

			if !yield(layer, nil) {
				return
			}
		}
	}
}
