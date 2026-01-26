package executable

import (
	"context"
	"errors"
	"iter"

	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/platforms"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/kubepkgspec/pkg/workload"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/empty"
	"github.com/octohelm/crkit/pkg/oci/mutate"
)

func WithImageName(imageName string) mutate.IndexMutatorOption {
	return func(m *mutate.IndexMutator) {
		m.Add(func(ctx context.Context, base oci.Index) (oci.Index, error) {
			image := workload.ParseImage(imageName)

			annotations := make(map[string]string)

			annotations[ocispecv1.AnnotationBaseImageName] = image.Name

			if image.Tag != "" {
				annotations[ocispecv1.AnnotationRefName] = image.Tag
			}

			d, err := base.Descriptor(ctx)
			if err != nil {
				return nil, err
			}

			if d.ArtifactType == "" {
				annotations[images.AnnotationImageName] = image.FullName()
			}

			return mutate.WithAnnotations(base, annotations)
		})
	}
}

func WithAnnotations(annotations map[string]string) mutate.IndexMutatorOption {
	return func(m *mutate.IndexMutator) {
		m.Add(func(ctx context.Context, base oci.Index) (oci.Index, error) {
			return mutate.WithAnnotations(base, annotations)
		})
	}
}

type Packer struct{}

func (p *Packer) PackAsIndex(ctx context.Context, blobs iter.Seq[oci.Blob], options ...mutate.IndexMutatorOption) (oci.Index, error) {
	idx, err := p.Pack(ctx, blobs)
	if err != nil {
		return nil, err
	}

	mut := &mutate.IndexMutator{}
	mut.Build(options...)

	idx, err = mut.Apply(ctx, idx)
	if err != nil {
		return nil, err
	}

	return mutate.AppendManifests(empty.Index, idx)
}

func (p *Packer) Pack(ctx context.Context, blobs iter.Seq[oci.Blob]) (oci.Index, error) {
	platformed := map[string][]oci.Blob{}
	noPlatformed := make([]oci.Blob, 0)

	for blob := range blobs {
		desc, err := blob.Descriptor(ctx)
		if err != nil {
			return nil, err
		}

		if desc.Platform == nil {
			noPlatformed = append(noPlatformed, blob)
			continue
		}

		pl := platforms.Format(*desc.Platform)

		platformed[pl] = append(platformed[pl], blob)
	}

	if len(platformed) == 0 {
		return nil, errors.New("at least on platformed executable")
	}

	idx := empty.Index

	for pl, platformedBlobs := range platformed {
		img, err := p.packAsPlatformedImage(ctx, pl, platformedBlobs...)
		if err != nil {
			return nil, err
		}

		if len(noPlatformed) > 0 {
			img, err = mutate.AppendLayers(img, noPlatformed...)
			if err != nil {
				return nil, err
			}
		}

		idx, err = mutate.AppendManifests(idx, img)
		if err != nil {
			return nil, err
		}
	}

	return mutate.WithArtifactType(idx, IndexArtifactType)
}

func (p *Packer) packAsPlatformedImage(ctx context.Context, platform string, blobs ...oci.Blob) (idx oci.Image, err error) {
	idx = empty.Image

	idx, err = mutate.WithPlatform(idx, platform)
	if err != nil {
		return
	}

	idx, err = mutate.AppendLayers(idx, blobs...)
	if err != nil {
		return
	}

	idx, err = mutate.WithArtifactType(idx, ArtifactType)
	if err != nil {
		return
	}

	return
}
