package fs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

type blobStore struct {
	workspace *workspace
}

var _ content.DigestIterable = &blobStore{}

func (bs *blobStore) Digests(ctx context.Context) iter.Seq2[digest.Digest, error] {
	return func(yield func(digest.Digest, error) bool) {
		yieldNamed := func(named digest.Digest, err error) bool {
			return yield(named, err)
		}

		err := bs.workspace.WalkDir(ctx, bs.workspace.layout.BlobsPath(), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if path == "." || d.IsDir() {
				return nil
			}

			dir, base := filepath.Split(path)
			if base != "data" {
				return nil
			}

			parentDir, hex := filepath.Split(strings.TrimSuffix(dir, string(filepath.Separator)))
			alg := filepath.Dir(strings.TrimSuffix(parentDir, string(filepath.Separator)))

			dgst := digest.NewDigestFromHex(alg, hex)
			if err := dgst.Validate(); err != nil {
				return fmt.Errorf("invalid digest of data path %s: %w", path, err)
			}
			
			if !yieldNamed(dgst, nil) {
				return fs.SkipAll
			}

			return nil
		})
		if err != nil {
			if !yield("", err) {
				return
			}
		}
	}
}

func (bs *blobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return bs.workspace.Delete(ctx, bs.workspace.layout.BlobDataPath(dgst))
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
	file, err := bs.workspace.Reader(ctx, bs.workspace.layout.BlobDataPath(dgst))
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

	if err := bs.workspace.PutContent(ctx, bs.workspace.layout.UploadStartedAtPath(id), []byte(startedAt.Format(time.RFC3339))); err != nil {
		return nil, err
	}

	uploadDataPath := bs.workspace.layout.UploadDataPath(id)

	fileWriter, err := bs.workspace.Writer(ctx, uploadDataPath, false)
	if err != nil {
		return nil, err
	}

	return &blobWriter{
		ctx:       ctx,
		id:        id,
		startedAt: startedAt,
		workspace: bs.workspace,
		resumable: true,

		path:       uploadDataPath,
		fileWriter: fileWriter,
		digester:   digest.SHA256.Digester(),
	}, nil
}

func (bs *blobStore) Resume(ctx context.Context, id string) (content.BlobWriter, error) {
	startedAtBytes, err := bs.workspace.GetContent(ctx, bs.workspace.layout.UploadStartedAtPath(id))
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

	fileWriter, err := bs.workspace.Writer(ctx, uploadDataPath, true)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrBlobUploadUnknown{}
		}
		return nil, err
	}

	b := &blobWriter{
		ctx:       ctx,
		id:        id,
		startedAt: startedAt,
		workspace: bs.workspace,
		resumable: true,

		path:       uploadDataPath,
		digester:   digest.SHA256.Digester(),
		fileWriter: fileWriter,
	}

	if err := b.resumeDigestIfNeed(ctx); err != nil {
		return nil, err
	}

	return b, nil
}
