package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"
)

func NewBlobStore(fsys filesystem.FileSystem) content.BlobStore {
	return &blobStore{fs: fsys}
}

type blobStore struct {
	fs filesystem.FileSystem
}

func (f *blobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return f.fs.RemoveAll(ctx, defaultLayout.BlobDataPath(dgst))
}

func (f *blobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	s, err := f.fs.Stat(ctx, defaultLayout.BlobDataPath(dgst))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrManifestBlobUnknown{
				Digest: dgst,
			}
		}
		return nil, err
	}
	return &manifestv1.Descriptor{
		Digest: dgst,
		Size:   s.Size(),
	}, nil
}

func (f *blobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	file, err := f.fs.OpenFile(ctx, defaultLayout.BlobDataPath(dgst), os.O_RDONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrManifestBlobUnknown{
				Digest: dgst,
			}
		}
		return nil, err
	}
	return file, nil
}

func (f *blobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	id := uuid.New().String()

	uploadPath := defaultLayout.UploadDataPath(id)

	if err := filesystem.MkdirAll(ctx, f.fs, filepath.Dir(uploadPath)); err != nil {
		return nil, err
	}

	file, err := f.fs.OpenFile(ctx, uploadPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &writer{
		id:       id,
		path:     uploadPath,
		digester: digest.SHA256.Digester(),
		file:     file,
		store:    f,
	}, nil
}

type writer struct {
	id       string
	digester digest.Digester
	file     filesystem.File
	path     string

	store  *blobStore
	offset int64
	once   sync.Once

	err error
}

func (w *writer) ID() string {
	return w.id
}

func (w *writer) Write(p []byte) (n int, err error) {
	n, err = w.file.Write(p)
	w.digester.Hash().Write(p[:n])
	w.offset += int64(len(p))
	return n, err
}

func (w *writer) Close() error {
	w.once.Do(func() {
		w.err = w.file.Close()
	})

	return w.err
}

func (w *writer) Digest(ctx context.Context) digest.Digest {
	return w.digester.Digest()
}

func (w *writer) Size(ctx context.Context) int64 {
	return w.offset
}

func (w *writer) Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	defer func() {
		_ = w.store.fs.RemoveAll(ctx, filepath.Dir(w.path))
	}()

	if err := w.Close(); err != nil {
		return nil, err
	}

	size := w.Size(ctx)
	dgst := w.Digest(ctx)

	if expected.Size > 0 && expected.Size != size {
		return nil, errors.Wrapf(content.ErrBlobInvalidLength, "unexpected commit size %d, expected %d", size, expected.Size)
	}

	if expected.Digest != "" && expected.Digest != dgst {
		return nil, &content.ErrBlobInvalidDigest{
			Digest: dgst,
			Reason: fmt.Errorf("not match %s", expected.Digest),
		}
	}

	target := defaultLayout.BlobDataPath(dgst)

	_, err := w.store.fs.Stat(ctx, target)
	if err == nil {
		// remove uploaded
		return &manifestv1.Descriptor{
			Size:      size,
			Digest:    dgst,
			MediaType: expected.MediaType,
		}, nil
	}

	if err := filesystem.MkdirAll(ctx, w.store.fs, filepath.Dir(target)); err != nil {
		return nil, err
	}

	if err := w.store.fs.Rename(ctx, w.path, target); err != nil {
		return nil, err
	}

	return &manifestv1.Descriptor{
		Size:      size,
		Digest:    dgst,
		MediaType: expected.MediaType,
	}, nil
}
