package fs

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"
)

type blobWriter struct {
	ctx context.Context

	id        string
	startedAt time.Time

	digester       digest.Digester
	uploadDataPath string
	file           filesystem.File

	workspace *workspace

	written   int64
	resumable bool

	closeOnce sync.Once
	err       error
}

func (bw *blobWriter) ID() string {
	return bw.id
}

func (bw *blobWriter) Write(p []byte) (n int, err error) {
	if err := bw.resumeDigestIfNeed(bw.ctx); err != nil {
		return 0, err
	}

	n, err = bw.file.Write(p)
	bw.digester.Hash().Write(p[:n])
	bw.written += int64(len(p))

	return n, err
}

func (bw *blobWriter) Close() error {
	bw.closeOnce.Do(func() {
		if err := bw.file.Close(); err != nil {
			bw.err = err
			return
		}

		if err := bw.storeHashState(bw.ctx); err != nil {
			bw.err = err
			return
		}
	})
	return bw.err
}

func (bw *blobWriter) Digest(ctx context.Context) digest.Digest {
	return bw.digester.Digest()
}

func (bw *blobWriter) Size(ctx context.Context) int64 {
	return bw.written
}

func (bw *blobWriter) Cancel(ctx context.Context) error {
	if err := bw.Close(); err != nil {
		return err
	}

	return bw.workspace.Remove(ctx, filepath.Dir(bw.uploadDataPath))
}

func (bw *blobWriter) Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	if err := bw.resumeDigestIfNeed(ctx); err != nil {
		return nil, err
	}

	if err := bw.Close(); err != nil {
		return nil, err
	}

	defer func() {
		// remove full uploaded
		_ = bw.workspace.Remove(ctx, filepath.Dir(bw.uploadDataPath))
	}()

	size := bw.Size(ctx)
	dgst := bw.Digest(ctx)

	if expected.Size > 0 && expected.Size != size {
		return nil, &content.ErrBlobInvalidLength{
			Reason: fmt.Sprintf("unexpected commit size %d, expected %d", size, expected.Size),
		}
	}

	if expected.Digest != "" && expected.Digest != dgst {
		return nil, &content.ErrBlobInvalidDigest{
			Digest: dgst,
			Reason: fmt.Errorf("not match %s", expected.Digest),
		}
	}

	blobDataPath := bw.workspace.layout.BlobDataPath(dgst)

	// skip moving when digest exists
	if _, err := bw.workspace.Stat(ctx, blobDataPath); err == nil {
		return &manifestv1.Descriptor{
			Size:      size,
			Digest:    dgst,
			MediaType: expected.MediaType,
		}, nil
	}

	if err := bw.workspace.Move(ctx, bw.uploadDataPath, blobDataPath); err != nil {
		return nil, err
	}

	return &manifestv1.Descriptor{
		Size:      size,
		Digest:    dgst,
		MediaType: expected.MediaType,
	}, nil
}
