package empty

import (
	"context"
	"iter"
	"sync"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

var emptyConfig = partial.BlobFromBytes(
	[]byte(`{}`),
	ocispecv1.Descriptor{
		MediaType: ocispecv1.MediaTypeEmptyJSON,
	},
)

var Image oci.Image = &image{}

type image struct {
	next internal.Image
	err  error
	once sync.Once
}

func (i *image) init(ctx context.Context) error {
	return i.next.Build(func(m *ocispecv1.Manifest) error {
		return i.next.CollectTo(ctx, i, m)
	})
}

func (i *image) initOnce(ctx context.Context) {
	i.once.Do(func() {
		if err := i.init(ctx); err != nil {
			i.err = err
			return
		}
	})
}

func (i *image) Config(ctx context.Context) (oci.Blob, error) {
	return emptyConfig, nil
}

func (i *image) Layers(ctx context.Context) iter.Seq2[oci.Blob, error] {
	return func(yield func(oci.Blob, error) bool) {
	}
}

func (i *image) Descriptor(ctx context.Context) (desc ocispecv1.Descriptor, err error) {
	i.initOnce(ctx)

	desc, err = i.next.Descriptor(ctx)
	err = i.err

	return
}

func (i *image) Value(ctx context.Context) (img ocispecv1.Manifest, err error) {
	i.initOnce(ctx)

	img, err = i.next.Value(ctx)
	err = i.err

	return
}

func (i *image) Raw(ctx context.Context) (raw []byte, err error) {
	i.initOnce(ctx)

	raw, err = i.next.Raw(ctx)
	err = i.err

	return
}
