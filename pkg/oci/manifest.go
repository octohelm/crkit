package oci

import (
	"context"
	"io"
	"iter"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Manifest interface {
	Descriptor(ctx context.Context) (ocispecv1.Descriptor, error)
	Raw(ctx context.Context) ([]byte, error)
}

type Index interface {
	Manifest

	Value(ctx context.Context) (ocispecv1.Index, error)
	Manifests(ctx context.Context) iter.Seq2[Manifest, error]
}

type Image interface {
	Manifest

	Value(ctx context.Context) (ocispecv1.Manifest, error)
	Config(ctx context.Context) (Blob, error)
	Layers(ctx context.Context) iter.Seq2[Blob, error]
}

type Blob interface {
	Descriptor(ctx context.Context) (ocispecv1.Descriptor, error)
	Open(ctx context.Context) (io.ReadCloser, error)
}
