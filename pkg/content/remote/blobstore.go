package remote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

var _ content.BlobStore = &blobStore{}

type blobStore struct {
	*repository
}

func (b *blobStore) normalizeError(err error) error {
	terr := &transport.Error{}
	if errors.As(err, &terr) {
		if terr.StatusCode == http.StatusNotFound {
			return content.ErrBlobUnknown
		}
	}
	return err
}

func (b *blobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	return errors.New("not supported")
}

func (b *blobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	dd, err := b.puller.Layer(ctx, b.repo.Digest(dgst.String()))
	if err != nil {
		return nil, b.normalizeError(err)
	}

	d, err := partial.Descriptor(dd)
	if err != nil {
		return nil, b.normalizeError(err)
	}

	return &manifestv1.Descriptor{
		MediaType:   string(d.MediaType),
		Digest:      digest.NewDigestFromHex(d.Digest.Algorithm, d.Digest.Hex),
		Size:        d.Size,
		Annotations: d.Annotations,
	}, nil
}

func (b *blobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadCloser, error) {
	d, err := b.puller.Layer(ctx, b.repo.Digest(dgst.String()))
	if err != nil {
		return nil, b.normalizeError(err)
	}

	r, err := d.Compressed()
	if err != nil {
		return nil, b.normalizeError(err)
	}

	return &nopSeeker{ReadCloser: r}, nil
}

func (b *blobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	r, w := io.Pipe()

	bw := &blobWriter{
		ctx:       ctx,
		b:         b,
		r:         r,
		w:         w,
		startedAt: time.Now(),
	}

	return bw, nil
}

type nopSeeker struct {
	io.ReadCloser
}

func (nopSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

var _ content.BlobWriter = &blobWriter{}

type blobWriter struct {
	ctx context.Context
	b   *blobStore

	layer v1.Layer

	r *io.PipeReader
	w *io.PipeWriter

	*sizedDigestWriter

	startedAt time.Time
	err       error

	once   sync.Once
	wg     sync.WaitGroup
	cancel func()
}

func (b *blobWriter) Close() error {
	return b.w.Close()
}

func (b *blobWriter) Size(ctx context.Context) int64 {
	b.initIfNeed()

	return b.sizedDigestWriter.written
}

func (b *blobWriter) Digest(ctx context.Context) digest.Digest {
	b.initIfNeed()

	return b.sizedDigestWriter.Digest()
}

func (b *blobWriter) initIfNeed() {
	b.once.Do(func() {
		b.sizedDigestWriter = &sizedDigestWriter{
			Digester: digest.SHA256.Digester(),
			Writer:   b.w,
		}

		ctx, cancel := context.WithCancel(b.ctx)
		b.cancel = cancel

		b.wg.Add(1)
		go func() {
			defer b.wg.Done()

			if err := b.b.pusher.Upload(ctx, b.b.repo, b.AsLayer()); err != nil {
				b.err = err
			}
		}()
	})
}

func (b *blobWriter) ReadFrom(r io.Reader) (n int64, err error) {
	b.initIfNeed()

	return io.Copy(b.sizedDigestWriter, r)
}

func (b *blobWriter) Write(p []byte) (int, error) {
	b.initIfNeed()

	return b.sizedDigestWriter.Write(p)
}

func (b *blobWriter) StartedAt() time.Time {
	return b.startedAt
}

func (b *blobWriter) ID() string {
	return fmt.Sprintf("%d", b.startedAt.Unix())
}

func (b *blobWriter) Commit(ctx context.Context, expect manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	if err := b.Close(); err != nil {
		return nil, err
	}

	b.wg.Wait()

	if err := b.err; err != nil {
		return nil, err
	}

	d := &manifestv1.Descriptor{}
	d.Size = b.Size(ctx)
	d.Digest = b.Digest(ctx)

	if expect.Size > 0 {
		if d.Size != expect.Size {
			return nil, errors.Wrapf(content.ErrBlobInvalidLength, "expect %d, but got %d", expect.Size, d.Size)
		}
	}

	if expect.Digest != "" {
		if d.Digest != expect.Digest {
			return nil, &content.ErrBlobInvalidDigest{
				Digest: d.Digest,
				Reason: fmt.Errorf("not match %s", expect.Digest),
			}
		}
	}

	return d, nil
}

func (b *blobWriter) Cancel(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}
	return b.Close()
}

func (b *blobWriter) AsLayer() v1.Layer {
	return &proxyLayer{
		b: b,
	}
}

type proxyLayer struct {
	b *blobWriter
}

func (a *proxyLayer) Digest() (v1.Hash, error) {
	d := a.b.Digest(context.Background())
	return v1.Hash{Algorithm: string(d.Algorithm()), Hex: d.Hex()}, nil
}

func (a *proxyLayer) Size() (int64, error) {
	return a.b.Size(context.Background()), nil
}

func (a *proxyLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (a *proxyLayer) MediaType() (types.MediaType, error) {
	return types.DockerLayer, nil
}

func (a *proxyLayer) Uncompressed() (io.ReadCloser, error) {
	return a.b.r, nil
}

func (a *proxyLayer) Compressed() (io.ReadCloser, error) {
	return a.Uncompressed()
}

type sizedDigestWriter struct {
	io.Writer
	digest.Digester
	written int64
}

func (s *sizedDigestWriter) Write(p []byte) (n int, err error) {
	if _, err := s.Hash().Write(p); err != nil {
		return 0, err
	}

	n, err = s.Writer.Write(p)
	if err != nil {
		return 0, err
	}
	s.written += int64(n)

	return
}
