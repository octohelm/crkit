package persist

import (
	"context"
	"github.com/opencontainers/go-digest"
	"io"
	"os"
	"path"
	"sync"

	"github.com/google/uuid"
	"github.com/octohelm/crkit/pkg/distribution"
	"github.com/octohelm/unifs/pkg/filesystem"
)

type blobService struct {
	fsys filesystem.FileSystem
}

func (s *blobService) uploadFilename(id string) (string, string, error) {
	if id == "" {
		uid, err := uuid.NewV7()
		if err != nil {
			return id, "", err
		}
		id = uid.String()
	}

	// uploads/<uuid>/data
	return id, path.Join("uploads", id, "data"), nil
}

func (s *blobService) blobFilename(dgst distribution.Digest) string {
	// blobs/<alg>/<hash>/data
	return path.Join("blobs", dgst.Algorithm().String(), dgst.Hex(), "data")
}

func (s *blobService) Stat(ctx context.Context, dgst distribution.Digest) (distribution.Descriptor, error) {
	stat, err := s.fsys.Stat(ctx, s.blobFilename(dgst))
	if err != nil {
		return distribution.Descriptor{}, err
	}
	return distribution.Descriptor{
		Digest: dgst,
		Size:   stat.Size(),
	}, nil
}

func (s *blobService) Open(ctx context.Context, dgst distribution.Digest) (io.ReadSeekCloser, error) {
	return s.fsys.OpenFile(ctx, s.blobFilename(dgst), os.O_RDONLY, os.ModePerm)
}

func (s *blobService) Delete(ctx context.Context, dgst distribution.Digest) error {
	return s.fsys.RemoveAll(ctx, s.blobFilename(dgst))
}

func (s *blobService) Create(ctx context.Context) (distribution.BlobWriter, error) {
	id, uploadFile, err := s.uploadFilename("")
	if err != nil {
		return nil, err
	}

	if err := filesystem.MkdirAll(ctx, s.fsys, path.Dir(uploadFile)); err != nil {
		return nil, err
	}

	file, err := filesystem.Create(ctx, s.fsys, uploadFile)
	if err != nil {
		return nil, err
	}

	return &blobWriter{
		blobService:      s,
		id:               id,
		uploadedFilename: uploadFile,
		digester:         digest.SHA256.Digester(),
		file:             file,
	}, nil
}

func (s *blobService) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	id, uploadFile, err := s.uploadFilename(id)
	if err != nil {
		return nil, err
	}

	info, err := s.fsys.Stat(ctx, uploadFile)
	if err != nil {
		return nil, err
	}

	file, err := s.fsys.OpenFile(ctx, uploadFile, os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &blobWriter{
		blobService:      s,
		id:               id,
		uploadedFilename: uploadFile,
		file:             file,
		written:          info.Size(),
	}, nil
}

type blobWriter struct {
	blobService *blobService

	id               string
	uploadedFilename string
	file             filesystem.File

	digester digest.Digester
	written  int64

	once sync.Once
}

func (b *blobWriter) Write(p []byte) (n int, err error) {
	defer func() {
		b.written += int64(n)
	}()
	_, _ = b.digester.Hash().Write(p)
	return b.file.Write(p)
}

func (b *blobWriter) Size() int64 {
	return b.written
}

func (b *blobWriter) ID() string {
	return b.id
}

func (b *blobWriter) Cancel(ctx context.Context) error {
	_ = b.Close()
	return b.blobService.fsys.RemoveAll(ctx, b.uploadedFilename)
}

func (b *blobWriter) Commit(ctx context.Context, provisional distribution.Descriptor) (distribution.Descriptor, error) {
	_ = b.Close()

	dgst := b.digester.Digest()

	if provisional.Digest != "" && provisional.Digest != dgst {
		return distribution.Descriptor{}, &distribution.ErrDigestNotMatch{
			Expect: provisional.Digest,
			Actual: dgst,
		}
	}

	if err := b.blobService.fsys.Rename(ctx, b.uploadedFilename, b.blobService.blobFilename(dgst)); err != nil {
		return distribution.Descriptor{}, err
	}

	return distribution.Descriptor{
		MediaType: provisional.MediaType,
		Digest:    dgst,
		Size:      b.Size(),
	}, nil
}

func (b *blobWriter) Close() error {
	b.once.Do(func() {
		_ = b.file.Close()
	})
	return nil
}
