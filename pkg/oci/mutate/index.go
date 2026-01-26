package mutate

import (
	"cmp"
	"context"
	"iter"
	"maps"
	"sync"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
)

func AppendManifests(base oci.Index, manifests ...oci.Manifest) (oci.Index, error) {
	if len(manifests) > 0 {
		return &index{Index: base, manifests: manifests}, nil
	}
	return base, nil
}

type index struct {
	oci.Index

	next internal.Index

	artifactType string
	annotations  map[string]string

	manifests []oci.Manifest

	err  error
	once sync.Once
}

func (i *index) init(ctx context.Context) error {
	return i.next.Build(func(m *ocispecv1.Index) error {
		base, err := i.Index.Value(ctx)
		if err != nil {
			return err
		}

		m.ArtifactType = cmp.Or(i.artifactType, base.ArtifactType)

		if len(base.Annotations) > 0 {
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			maps.Copy(m.Annotations, base.Annotations)
		}

		if len(i.annotations) > 0 {
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			maps.Copy(m.Annotations, i.annotations)
		}

		return i.next.CollectTo(ctx, i, m)
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
		for l, err := range i.Index.Manifests(ctx) {
			if !yield(l, err) {
				return
			}
		}

		if len(i.manifests) > 0 {
			for _, m := range i.manifests {
				if !yield(m, nil) {
					return
				}
			}
		}
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
