package distribution

import (
	"context"
	"io"
)

type BlobStatter interface {
	Stat(ctx context.Context, dgst Digest) (Descriptor, error)
}

type BlobDeleter interface {
	Delete(ctx context.Context, dgst Digest) error
}

type BlobProvider interface {
	Open(ctx context.Context, dgst Digest) (io.ReadSeekCloser, error)
}

type BlobIngester interface {
	Create(ctx context.Context) (BlobWriter, error)
	Resume(ctx context.Context, id string) (BlobWriter, error)
}

type BlobWriter interface {
	io.WriteCloser

	Size() int64
	ID() string

	Cancel(ctx context.Context) error
	Commit(ctx context.Context, provisional Descriptor) (canonical Descriptor, err error)
}

type BlobService interface {
	BlobStatter
	BlobProvider
	BlobDeleter
	BlobIngester
}
