package fs

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/opencontainers/go-digest"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
)

type blobWriter struct {
	ctx context.Context

	id        string
	startedAt time.Time

	digester   digest.Digester
	fileWriter driver.FileWriter
	path       string

	written int64

	workspace *workspace

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

	n, err = bw.fileWriter.Write(p)
	bw.digester.Hash().Write(p[:n])
	bw.written += int64(len(p))

	return n, err
}

func (bw *blobWriter) Digest(ctx context.Context) digest.Digest {
	return bw.digester.Digest()
}

func (bw *blobWriter) Size(ctx context.Context) int64 {
	return bw.fileWriter.Size()
}

func (bw *blobWriter) Cancel(ctx context.Context) error {
	return bw.fileWriter.Cancel(ctx)
}

func (bw *blobWriter) Close() error {
	bw.closeOnce.Do(func() {
		if err := bw.fileWriter.Close(); err != nil {
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

func (bw *blobWriter) Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	if err := bw.fileWriter.Commit(ctx); err != nil {
		return nil, err
	}

	if err := bw.Close(); err != nil {
		return nil, err
	}

	defer func() {
		// remove full uploaded
		_ = bw.cleanUpload(ctx)
	}()

	desc := &manifestv1.Descriptor{
		Size:      bw.Size(ctx),
		Digest:    bw.Digest(ctx),
		MediaType: expected.MediaType,
	}

	if expected.Size > 0 && expected.Size != desc.Size {
		return nil, &content.ErrBlobInvalidLength{
			Reason: fmt.Sprintf("unexpected commit size %d, expected %d", desc.Size, expected.Size),
		}
	}

	if expected.Digest != "" && expected.Digest != desc.Digest {
		return nil, &content.ErrBlobInvalidDigest{
			Digest: desc.Digest,
			Reason: fmt.Errorf("not match %s", expected.Digest),
		}
	}

	if err := bw.moveBlob(ctx, desc); err != nil {
		return nil, err
	}

	return desc, nil
}

func (bw *blobWriter) cleanUpload(ctx context.Context) error {
	return bw.workspace.Delete(ctx, path.Dir(bw.path))
}

func (bw *blobWriter) moveBlob(ctx context.Context, desc *manifestv1.Descriptor) error {
	blobDataPath := bw.workspace.layout.BlobDataPath(desc.Digest)

	// skip moving when digest exists
	if _, err := bw.workspace.Stat(ctx, blobDataPath); err == nil {
		return nil
	}

	return bw.workspace.Move(ctx, bw.path, blobDataPath)
}
