package mutate

import (
	"cmp"
	"context"
	"iter"
	"maps"
	"sync"

	"github.com/containerd/platforms"
	"github.com/go-json-experiment/json"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func WithImageConfig(base oci.Image, imageConfig *ocispecv1.ImageConfig) (oci.Image, error) {
	imageConfigRaw, err := json.Marshal(imageConfig, json.Deterministic(true))
	if err != nil {
		return nil, err
	}

	return WithConfig(
		base,
		partial.BlobFromBytes(imageConfigRaw, ocispecv1.Descriptor{MediaType: ocispecv1.MediaTypeImageConfig}),
	)
}

func WithConfig(base oci.Image, config oci.Blob) (oci.Image, error) {
	if config != nil {
		return &image{Image: base, config: config}, nil
	}
	return base, nil
}

func AppendLayers(base oci.Image, layers ...oci.Blob) (oci.Image, error) {
	if len(layers) > 0 {
		return &image{Image: base, layers: layers}, nil
	}
	return base, nil
}

func WithPlatform(base oci.Image, p string) (oci.Image, error) {
	if p != "" {
		pl, err := platforms.Parse(p)
		if err != nil {
			return nil, err
		}
		return &image{Image: base, platform: &pl}, nil
	}

	return base, nil
}

type image struct {
	oci.Image

	next internal.Image

	artifactType string
	platform     *ocispecv1.Platform
	annotations  map[string]string
	config       oci.Blob
	layers       []oci.Blob

	err  error
	once sync.Once
}

func (i *image) init(ctx context.Context) error {
	return i.next.Build(func(m *ocispecv1.Manifest) error {
		base, err := i.Image.Value(ctx)
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

		if err := i.next.CollectTo(ctx, i, m); err != nil {
			return err
		}

		if i.platform != nil {
			m.Config.Platform = i.platform
		} else {
			if base.Config.Platform != nil {
				m.Config.Platform = base.Config.Platform
			}
		}

		return nil
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

func (i *image) Config(ctx context.Context) (oci.Blob, error) {
	if i.config != nil {
		return i.config, nil
	}
	return i.Image.Config(ctx)
}

func (i *image) Layers(ctx context.Context) iter.Seq2[oci.Blob, error] {
	return func(yield func(oci.Blob, error) bool) {
		for l, err := range i.Image.Layers(ctx) {
			if !yield(l, err) {
				return
			}
		}

		if len(i.layers) > 0 {
			for _, l := range i.layers {
				if !yield(l, nil) {
					return
				}
			}
		}
	}
}
