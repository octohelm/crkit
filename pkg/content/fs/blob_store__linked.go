package fs

import (
	"context"
	"io"
	"os"

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
		errUnknownFunc: func(dgst digest.Digest) error {
			return &content.ErrBlobUnknown{
				Digest: dgst,
			}
		},
	}
}

func newLinkedBlobStoreAsManifestService(w *workspace, named reference.Named) *linkedBlobStore {
	return &linkedBlobStore{
		workspace: w,
		blobStore: &blobStore{workspace: w},
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

type linkedBlobStore struct {
	workspace      *workspace
	blobStore      *blobStore
	errUnknownFunc func(dgst digest.Digest) error
	linkPathFunc   func(dgst digest.Digest) string
}

func (lbs *linkedBlobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return lbs.workspace.Remove(ctx, lbs.linkPathFunc(dgst))
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

	if err := w.linkedBlobStore.workspace.PutContent(ctx, w.linkedBlobStore.linkPathFunc(d.Digest), []byte(d.Digest)); err != nil {
		return nil, err
	}

	return d, nil
}
