package distribution

import (
	"context"
	"github.com/opencontainers/go-digest"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ManifestService interface {
	Exists(ctx context.Context, dgst Digest) (bool, error)
	Get(ctx context.Context, dgst Digest) (Manifest, error)
	Delete(ctx context.Context, dgst Digest) error
	Put(ctx context.Context, manifest Manifest) (Digest, error)
}

type Digest = digest.Digest

type Descriptor = specv1.Descriptor

type Manifest interface {
	MediaType() string
	References() []Descriptor
	Payload() ([]byte, error)
}
