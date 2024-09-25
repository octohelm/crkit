package proxy

import (
	"context"
	"io"

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type proxyBlobStore struct {
	repositoryName reference.Named

	localStore  content.BlobStore
	remoteStore content.BlobStore
}

var _ content.BlobStore = &proxyBlobStore{}

func (pbs *proxyBlobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	return nil, errors.New("not implements")
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
		Closer: CloserFunc(func() error {
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

	if !errors.Is(err, content.ErrBlobUnknown) {
		return &manifestv1.Descriptor{}, err
	}

	d, err := pbs.remoteStore.Info(ctx, dgst)
	if err != nil {
		if errors.Is(err, content.ErrBlobUnknown) {
			// FIXME hack to use open to trigger remote sync
			// harbor will return 404 when stat, util digest full synced
			b, err := pbs.remoteStore.Open(ctx, dgst)
			if err != nil {
				return nil, err
			}

			bw, err := pbs.localStore.Writer(ctx)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(bw, b); err != nil {
				return nil, err
			}
			defer b.Close()
			if _, err := bw.Commit(ctx, manifestv1.Descriptor{Digest: dgst}); err != nil {
				return nil, err
			}
			// use local stat
			return pbs.localStore.Info(ctx, dgst)
		}

		return nil, err
	}
	return d, nil
}

func CloserFunc(close func() error) io.Closer {
	return &closer{close}
}

type closer struct {
	close func() error
}

func (c *closer) Close() error {
	return c.close()
}
