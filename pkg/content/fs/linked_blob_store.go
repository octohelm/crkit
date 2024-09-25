package fs

import (
	"context"
	"io"
	"os"

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"
)

type linkedBlobStore struct {
	named          reference.Named
	fs             filesystem.FileSystem
	blobStore      content.BlobStore
	errUnknownFunc func(named reference.Named, dgst digest.Digest) error
	linkPathFunc   func(named reference.Named, dgst digest.Digest) string
}

func (l *linkedBlobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	p := l.linkPathFunc(l.named, dgst)

	return l.fs.RemoveAll(ctx, p)
}

func (l *linkedBlobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	link := l.linkPathFunc(l.named, dgst)

	_, err := l.fs.Stat(ctx, link)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, l.errUnknownFunc(l.named, dgst)
		}
		return nil, err
	}

	return l.blobStore.Info(ctx, dgst)
}

func (l *linkedBlobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	link := l.linkPathFunc(l.named, dgst)

	_, err := l.fs.Stat(ctx, link)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, l.errUnknownFunc(l.named, dgst)
		}
		return nil, err
	}

	return l.blobStore.Open(ctx, dgst)
}

func (l *linkedBlobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	w, err := l.blobStore.Writer(ctx)
	if err != nil {
		return nil, err
	}

	return &linkedPathWriter{
		linkedBlobStore: l,
		BlobWriter:      w,
	}, nil
}

type linkedPathWriter struct {
	content.BlobWriter
	*linkedBlobStore
}

func (w *linkedPathWriter) Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	d, err := w.BlobWriter.Commit(ctx, expected)
	if err != nil {
		return nil, err
	}

	link := w.linkPathFunc(w.named, d.Digest)
	if err := writeFile(ctx, w.fs, link, []byte(d.Digest)); err != nil {
		return nil, err
	}
	return d, nil
}
