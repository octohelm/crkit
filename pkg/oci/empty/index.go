package empty

import (
	"context"
	"iter"
	"sync"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
)

var Index oci.Index = &index{}

type index struct {
	next internal.Index
	err  error
	once sync.Once
}

func (i *index) init(ctx context.Context) error {
	return i.next.Build(func(m *ocispecv1.Index) error {
		return nil
	})
}

func (i *index) initOnce(ctx context.Context) {
	i.once.Do(func() {
		if err := i.init(ctx); err != nil {
			i.err = err
			return
		}
	})
}

func (i *index) Manifests(ctx context.Context) iter.Seq2[oci.Manifest, error] {
	return func(yield func(oci.Manifest, error) bool) {
	}
}

func (i *index) Descriptor(ctx context.Context) (desc ocispecv1.Descriptor, err error) {
	i.initOnce(ctx)

	desc, err = i.next.Descriptor(ctx)
	err = i.err

	return
}

func (i *index) Value(ctx context.Context) (img ocispecv1.Index, err error) {
	i.initOnce(ctx)

	img, err = i.next.Value(ctx)
	err = i.err

	return
}

func (i *index) Raw(ctx context.Context) (raw []byte, err error) {
	i.initOnce(ctx)

	raw, err = i.next.Raw(ctx)
	err = i.err

	return
}
