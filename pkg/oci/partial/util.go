package partial

import (
	"context"
	"iter"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
)

func CollectChildDescriptors(ctx context.Context, m oci.Manifest) ([]ocispecv1.Descriptor, error) {
	descriptors := make([]ocispecv1.Descriptor, 0)

	for m, err := range AllChildDescriptors(ctx, m) {
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, m)
	}

	return descriptors, nil
}

func AllChildDescriptors(ctx context.Context, m oci.Manifest) iter.Seq2[ocispecv1.Descriptor, error] {
	return func(yield func(ocispecv1.Descriptor, error) bool) {
		switch x := m.(type) {
		case oci.Index:
			for m, err := range x.Manifests(ctx) {
				if err != nil {
					if !yield(ocispecv1.Descriptor{}, err) {
						return
					}
					return
				}
				for sub, err := range AllChildDescriptors(ctx, m) {
					if !yield(sub, err) {
						return
					}
				}

				if !yield(m.Descriptor(ctx)) {
					return
				}
			}

			return
		case oci.Image:
			c, err := x.Config(ctx)
			if err != nil {
				yield(ocispecv1.Descriptor{}, err)
				return
			}

			if !yield(c.Descriptor(ctx)) {
				return
			}

			for b, err := range x.Layers(ctx) {
				if err != nil {
					yield(ocispecv1.Descriptor{}, err)
					return
				}

				if !yield(b.Descriptor(ctx)) {
					return
				}
			}
		}
	}
}

func CollectImages(ctx context.Context, index oci.Index) ([]oci.Image, error) {
	images := make([]oci.Image, 0)

	for m, err := range AllImages(ctx, index) {
		if err != nil {
			return nil, err
		}
		images = append(images, m)
	}

	return images, nil
}

func AllImages(ctx context.Context, index oci.Index) iter.Seq2[oci.Image, error] {
	return func(yield func(oci.Image, error) bool) {
		for m, err := range index.Manifests(ctx) {
			if err != nil {
				if !yield(nil, err) {
					return
				}
				return
			}

			switch x := m.(type) {
			case oci.Image:
				if !yield(x, nil) {
					return
				}
			case oci.Index:
				for s, err := range AllImages(ctx, x) {
					if !yield(s, err) {
						return
					}
				}
			}
		}
	}
}
