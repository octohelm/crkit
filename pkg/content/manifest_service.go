package content

import (
	"context"

	"github.com/opencontainers/go-digest"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
)

type ManifestService interface {
	Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error)
	Get(ctx context.Context, dgst digest.Digest) (manifestv1.Manifest, error)
	Put(ctx context.Context, manifest manifestv1.Manifest) (digest.Digest, error)
	Delete(ctx context.Context, dgst digest.Digest) error
}
