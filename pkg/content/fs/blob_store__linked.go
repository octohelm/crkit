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

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

func newLinkedBlobStore(w *workspace, named reference.Named) *linkedBlobStore {
	return &linkedBlobStore{
		workspace: w,
		blobStore: &blobStore{
			workspace: w,
		},
		linkPathFunc: func(dgst digest.Digest) string {
			return w.layout.RepositoryLayerLinkPath(named, dgst)
		},
		linkDirFunc: func() string {
			return w.layout.RepositoryLayersPath(named)
		},
		errUnknownFunc: func(dgst digest.Digest) error {
			return &content.ErrBlobUnknown{
				Digest: dgst,
			}
		},
	}
}

func newLinkedBlobStoreForManifestService(w *workspace, named reference.Named) *linkedBlobStore {
	return &linkedBlobStore{
		workspace: w,
		blobStore: &blobStore{workspace: w},
		linkDirFunc: func() string {
			return w.layout.RepositoryManifestRevisionsPath(named)
		},
		linkPathFunc: func(dgst digest.Digest) string {
			return w.layout.RepositoryManifestRevisionLinkPath(named, dgst)
		},
		errUnknownFunc: func(dgst digest.Digest) error {
			return &content.ErrManifestUnknownRevision{
				Name:     named.Name(),
				Revision: dgst,
			}
		},
	}
}

func newLinkedBlobStoreForTagService(w *workspace, named reference.Named, tag string) *linkedBlobStore {
	return &linkedBlobStore{
		workspace: w,
		blobStore: &blobStore{workspace: w},
		linkDirFunc: func() string {
			return w.layout.RepositoryManifestTagIndexPath(named, tag)
		},
		linkPathFunc: func(dgst digest.Digest) string {
			return w.layout.RepositoryManifestTagIndexLinkPath(named, tag, dgst)
		},
		errUnknownFunc: func(dgst digest.Digest) error {
			return &content.ErrManifestUnknownRevision{
				Name:     named.Name(),
				Revision: dgst,
			}
		},
	}
}

type linkedBlobStore struct {
	workspace      *workspace
	blobStore      *blobStore
	errUnknownFunc func(dgst digest.Digest) error

	linkDirFunc  func() string
	linkPathFunc func(dgst digest.Digest) string
}

var _ content.LinkedDigestIterable = &linkedBlobStore{}

func (lbs *linkedBlobStore) LinkedDigests(ctx context.Context) iter.Seq2[content.LinkedDigest, error] {
	return func(yield func(content.LinkedDigest, error) bool) {
		yieldLinkedDigest := func(named content.LinkedDigest, err error) bool {
			return yield(named, err)
		}

		if err := lbs.workspace.WalkDir(ctx, lbs.linkDirFunc(), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if path == "." || d.IsDir() {
				return nil
			}

			// {alg}/{hex}/link
			dir, base := filepath.Split(path)
			if base != "link" {
				return nil
			}

			parentDir, hex := filepath.Split(strings.TrimSuffix(dir, string(filepath.Separator)))
			alg := filepath.Base(strings.TrimSuffix(parentDir, string(filepath.Separator)))

			dgst := digest.NewDigestFromHex(alg, hex)
			if err := dgst.Validate(); err != nil {
				return fmt.Errorf("invalid linked digest of link path %s: %w", path, err)
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			if !yieldLinkedDigest(content.LinkedDigest{
				Digest:  dgst,
				ModTime: info.ModTime(),
			}, nil) {
				return fs.SkipAll
			}

			return nil
		}); err != nil {
			// skip base dir not exists
			if os.IsNotExist(err) {
				return
			}

			if !yield(content.LinkedDigest{}, err) {
				return
			}
		}
	}
}

func (lbs *linkedBlobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return lbs.workspace.Delete(ctx, lbs.linkPathFunc(dgst))
}

func (lbs *linkedBlobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	link := lbs.linkPathFunc(dgst)

	_, err := lbs.workspace.Stat(ctx, link)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lbs.errUnknownFunc(dgst)
		}
		return nil, err
	}

	return lbs.blobStore.Info(ctx, dgst)
}

func (lbs *linkedBlobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	link := lbs.linkPathFunc(dgst)

	_, err := lbs.workspace.Stat(ctx, link)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lbs.errUnknownFunc(dgst)
		}
		return nil, err
	}

	return lbs.blobStore.Open(ctx, dgst)
}

func (lbs *linkedBlobStore) Resume(ctx context.Context, id string) (content.BlobWriter, error) {
	w, err := lbs.blobStore.Resume(ctx, id)
	if err != nil {
		return nil, err
	}

	return &linkedBlobWriter{
		linkedBlobStore: lbs,
		BlobWriter:      w,
	}, nil
}

func (lbs *linkedBlobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	w, err := lbs.blobStore.Writer(ctx)
	if err != nil {
		return nil, err
	}

	return &linkedBlobWriter{
		linkedBlobStore: lbs,
		BlobWriter:      w,
	}, nil
}

type linkedBlobWriter struct {
	content.BlobWriter

	linkedBlobStore *linkedBlobStore
}

func (w *linkedBlobWriter) Commit(ctx context.Context, expected manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	d, err := w.BlobWriter.Commit(ctx, expected)
	if err != nil {
		return nil, err
	}

	// always put link to fresh mod time
	if err := w.linkedBlobStore.workspace.PutContent(ctx, w.linkedBlobStore.linkPathFunc(d.Digest), []byte(d.Digest)); err != nil {
		return nil, err
	}

	return d, nil
}
