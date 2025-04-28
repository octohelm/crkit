package fs

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"
)

type blobStore struct {
	workspace *workspace
}

func (bs *blobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return bs.workspace.Remove(ctx, bs.workspace.layout.BlobDataPath(dgst))
}

func (bs *blobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	s, err := bs.workspace.Stat(ctx, bs.workspace.layout.BlobDataPath(dgst))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrBlobUnknown{
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

func (bs *blobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	file, err := bs.workspace.Open(ctx, bs.workspace.layout.BlobDataPath(dgst))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrBlobUnknown{
				Digest: dgst,
			}
		}
		return nil, err
	}
	return file, nil
}

func (bs *blobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	id := uuid.New().String()
	startedAt := time.Now().UTC()

	if err := bs.workspace.PutContent(ctx, bs.workspace.layout.UploadDataStartedAtPath(id), []byte(startedAt.Format(time.RFC3339))); err != nil {
		return nil, err
	}

	uploadDataPath := bs.workspace.layout.UploadDataPath(id)

	if err := filesystem.MkdirAll(ctx, bs.workspace.fs, filepath.Dir(uploadDataPath)); err != nil {
		return nil, err
	}

	file, err := bs.workspace.fs.OpenFile(ctx, uploadDataPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &blobWriter{
		ctx:            ctx,
		id:             id,
		startedAt:      startedAt,
		uploadDataPath: uploadDataPath,
		digester:       digest.SHA256.Digester(),
		file:           file,
		workspace:      bs.workspace,
		resumable:      true,
	}, nil
}

func (bs *blobStore) Resume(ctx context.Context, id string) (content.BlobWriter, error) {
	startedAtBytes, err := bs.workspace.GetContent(ctx, bs.workspace.layout.UploadDataStartedAtPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrBlobUploadUnknown{}
		}
		return nil, err
	}

	startedAt, err := time.Parse(time.RFC3339, string(startedAtBytes))
	if err != nil {
		return nil, err
	}

	uploadDataPath := bs.workspace.layout.UploadDataPath(id)

	file, err := bs.workspace.fs.OpenFile(ctx, uploadDataPath, os.O_WRONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrBlobUploadUnknown{}
		}
		return nil, err
	}

	b := &blobWriter{
		ctx:            ctx,
		id:             id,
		startedAt:      startedAt,
		uploadDataPath: uploadDataPath,
		digester:       digest.SHA256.Digester(),
		file:           file,
		workspace:      bs.workspace,
		resumable:      true,
	}

	if err := b.resumeDigestIfNeed(ctx); err != nil {
		return nil, err
	}

	return b, nil
}
