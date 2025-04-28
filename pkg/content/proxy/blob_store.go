package proxy

import (
	"context"
	"errors"
	"io"

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/x/ptr"
	"github.com/opencontainers/go-digest"
)

type proxyBlobStore struct {
	repositoryName reference.Named

	localStore  content.BlobStore
	remoteStore content.BlobStore
}

var _ content.BlobStore = &proxyBlobStore{}

func (pbs *proxyBlobStore) Resume(ctx context.Context, id string) (content.BlobWriter, error) {
	return pbs.remoteStore.Resume(ctx, id)
}

func (pbs *proxyBlobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	return pbs.remoteStore.Writer(ctx)
}

func (pbs *proxyBlobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return pbs.localStore.Remove(ctx, dgst)
}

func (pbs *proxyBlobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	blob, err := pbs.localStore.Open(ctx, dgst)
	if err == nil {
		return blob, nil
	}

	blob, err = pbs.remoteStore.Open(ctx, dgst)
	if err != nil {
		return nil, err
	}

	bw, err := pbs.localStore.Writer(ctx)
	if err != nil {
		return nil, err
	}

	rsc := &struct {
		io.Reader
		io.Closer
	}{
		Reader: io.TeeReader(blob, bw),
		Closer: closerFunc(func() error {
			defer func() {
				err = blob.Close()
			}()

			_, err = bw.Commit(ctx, manifestv1.Descriptor{
				Digest: dgst,
			})

			return err
		}),
	}

	return rsc, nil
}

func (pbs *proxyBlobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	desc, err := pbs.localStore.Info(ctx, dgst)
	if err == nil {
		return desc, err
	}

	if !errors.As(err, ptr.Ptr(&content.ErrBlobUnknown{})) && !errors.As(err, ptr.Ptr(&content.ErrManifestBlobUnknown{})) {
		return &manifestv1.Descriptor{}, err
	}

	return pbs.remoteStore.Info(ctx, dgst)
}

func closerFunc(close func() error) io.Closer {
	return &closer{close}
}

type closer struct {
	close func() error
}

func (c *closer) Close() error {
	return c.close()
}
