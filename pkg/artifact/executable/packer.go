package executable

import (
	"context"
	"io"

	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/platforms"
	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"

	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
	"github.com/octohelm/kubepkgspec/pkg/workload"

	"github.com/octohelm/crkit/pkg/artifact"
)

const (
	ArtifactType           = "application/vnd.executable+type"
	MediaTypeBinaryContent = "application/vnd.executable"
)

func PlatformedBinary(platform string, open func() (io.ReadCloser, error)) (LayerWithPlatform, error) {
	p, err := platforms.Parse(platform)
	if err != nil {
		return nil, err
	}

	l, err := artifact.FromOpener(MediaTypeBinaryContent, open)
	if err != nil {
		return nil, err
	}

	return &layerWithPlatform{
		Layer:    l,
		platform: p,
	}, nil
}

type layerWithPlatform struct {
	artifact.Layer
	platform specv1.Platform
}

func (l layerWithPlatform) Platform() *containerregistryv1.Platform {
	return &containerregistryv1.Platform{
		OS:           l.platform.OS,
		Architecture: l.platform.Architecture,
		Variant:      l.platform.Variant,
		OSVersion:    l.platform.OSVersion,
	}
}

type LayerWithPlatform interface {
	artifact.Layer
	Platform() *containerregistryv1.Platform
}

type Packer struct{}

type Option = func(m *mutate.IndexAddendum)

func WithImageName(imageName string) Option {
	return func(add *mutate.IndexAddendum) {
		image := workload.ParseImage(imageName)

		add.Annotations[specv1.AnnotationBaseImageName] = image.Name

		if image.Tag != "" {
			add.Annotations[specv1.AnnotationRefName] = image.Tag
		}

		if add.ArtifactType == "" {
			add.Annotations[images.AnnotationImageName] = image.FullName()
		}
	}
}

func WithAnnotations(annotations map[string]string) Option {
	return func(add *mutate.IndexAddendum) {
		for k, v := range annotations {
			add.Annotations[k] = v
		}
	}
}

func (p *Packer) PackAsIndexOfOciTar(ctx context.Context, layers []LayerWithPlatform, options ...Option) (containerregistryv1.ImageIndex, error) {
	index, err := p.PackAsIndex(ctx, layers...)
	if err != nil {
		return nil, err
	}

	add := &mutate.IndexAddendum{
		Add: index,
	}

	d, err := partial.Descriptor(index)
	if err != nil {
		return nil, err
	}
	add.Descriptor = *d

	if add.Annotations == nil {
		add.Annotations = map[string]string{}
	}

	for _, opt := range options {
		opt(add)
	}

	return mutate.AppendManifests(empty.Index, *add), nil
}

func (p *Packer) PackAsIndex(ctx context.Context, layers ...LayerWithPlatform) (containerregistryv1.ImageIndex, error) {
	var index containerregistryv1.ImageIndex = empty.Index

	for _, layer := range layers {
		img, err := p.packAsImage(ctx, layer)
		if err != nil {
			return nil, err
		}

		d, err := partial.Descriptor(img)
		if err != nil {
			return nil, err
		}

		add := mutate.IndexAddendum{
			Add:        img,
			Descriptor: *d,
		}
		add.Platform = layer.Platform()

		index = mutate.AppendManifests(index, add)
	}

	return index, nil
}

func (p *Packer) packAsImage(ctx context.Context, layer LayerWithPlatform) (containerregistryv1.Image, error) {
	i := empty.Image

	gzipped, err := artifact.Gzipped(layer)
	if err != nil {
		return nil, err
	}

	i, err = mutate.AppendLayers(i, gzipped)
	if err != nil {
		return nil, err
	}

	return artifact.Artifact(i, artifact.EmptyConfig(ArtifactType))
}

func (p *Packer) appendManifests(idx containerregistryv1.ImageIndex, source partial.Describable, desc *containerregistryv1.Descriptor, image *kubepkgv1alpha1.Image) (containerregistryv1.ImageIndex, error) {
	if desc == nil {
		d, err := partial.Descriptor(source)
		if err != nil {
			return nil, err
		}
		desc = d
	}

	add := mutate.IndexAddendum{
		Add:        source,
		Descriptor: *desc,
	}

	if image != nil {
		if add.Annotations == nil {
			add.Annotations = map[string]string{}
		}

		if image.Name != "" {
			add.Annotations[specv1.AnnotationBaseImageName] = image.Name

			if add.ArtifactType == "" {
				add.Annotations[images.AnnotationImageName] = image.FullName()
			}
		}

		if image.Tag != "" {
			add.Annotations[specv1.AnnotationRefName] = image.Tag
		}
	}

	return mutate.AppendManifests(idx, add), nil
}
