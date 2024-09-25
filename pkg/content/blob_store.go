package content

import (
	"context"
	"io"
	"iter"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/opencontainers/go-digest"
)

type BlobStore interface {
	Ingester
	Provider
	Remover
}

type Provider interface {
	Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error)
	Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error)
}

type Ingester interface {
	Writer(ctx context.Context) (BlobWriter, error)
}

type Remover interface {
	Remove(ctx context.Context, dgst digest.Digest) error
}

type BlobLister interface {
	Blob(ctx context.Context) iter.Seq[digest.Digest]
}

type BlobWriter interface {
	io.WriteCloser
	ID() string
	Digest(ctx context.Context) digest.Digest
	Size(ctx context.Context) int64
	Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error)
}
