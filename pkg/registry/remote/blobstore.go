package remote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/distribution/distribution/v3"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type blobStore struct {
	*repository
}

func (b *blobStore) normalizeError(err error) error {
	terr := &transport.Error{}
	if errors.As(err, &terr) {
		if terr.StatusCode == http.StatusNotFound {
			return distribution.ErrBlobUnknown
		}
	}
	return nil
}

func (b *blobStore) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	return nil, errors.New("not supported")
}

func (b *blobStore) Delete(ctx context.Context, dgst digest.Digest) error {
	return errors.New("not supported")
}

func (b *blobStore) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	dd, err := b.puller.Layer(ctx, b.repo.Digest(dgst.String()))
	if err != nil {
		return distribution.Descriptor{}, b.normalizeError(err)
	}

	d, err := partial.Descriptor(dd)
	if err != nil {
		return distribution.Descriptor{}, b.normalizeError(err)
	}

	return distribution.Descriptor{
		MediaType:   string(d.MediaType),
		Digest:      digest.NewDigestFromHex(d.Digest.Algorithm, d.Digest.Hex),
		Size:        d.Size,
		Annotations: d.Annotations,
	}, nil
}

func (b *blobStore) Open(ctx context.Context, dgst digest.Digest) (io.ReadSeekCloser, error) {
	d, err := b.puller.Layer(ctx, b.repo.Digest(dgst.String()))
	if err != nil {
		return nil, b.normalizeError(err)
	}

	r, err := d.Compressed()
	if err != nil {
		return nil, err
	}

	return &nopSeeker{ReadCloser: r}, nil
}

type nopSeeker struct {
	io.ReadCloser
}

func (nopSeeker) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (b *blobStore) Get(ctx context.Context, dgst digest.Digest) ([]byte, error) {
	r, err := b.Open(ctx, dgst)
	if err != nil {
		return nil, b.normalizeError(err)
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (b *blobStore) Put(ctx context.Context, mediaType string, p []byte) (distribution.Descriptor, error) {
	r, err := b.Create(ctx)
	if err != nil {
		return distribution.Descriptor{}, nil
	}
	if _, err := r.Write(p); err != nil {
		return distribution.Descriptor{}, err
	}
	return r.Commit(ctx, distribution.Descriptor{})
}

func (b blobStore) ServeBlob(ctx context.Context, w http.ResponseWriter, r *http.Request, dgst digest.Digest) error {
	d, err := b.Stat(ctx, dgst)
	if err != nil {
		return err
	}

	setResponseHeaders(w, d)

	blobReader, err := b.Open(ctx, dgst)
	if err != nil {
		return err
	}
	defer func() {
		_ = blobReader.Close()
	}()

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, blobReader)

	return nil
}

func setResponseHeaders(w http.ResponseWriter, descriptor distribution.Descriptor) {
	w.Header().Set("Content-Length", strconv.FormatInt(descriptor.Size, 10))
	w.Header().Set("Content-Type", descriptor.MediaType)
	w.Header().Set("Docker-Content-Digest", descriptor.Digest.String())
	w.Header().Set("Etag", descriptor.Digest.String())
}

func (b *blobStore) Create(ctx context.Context, options ...distribution.BlobCreateOption) (distribution.BlobWriter, error) {
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

func (b *blobWriter) Size() int64 {
	return b.sizedDigestWriter.written
}

func (b *blobWriter) Digest() digest.Digest {
	return b.sizedDigestWriter.Digest()
}

func (b *blobWriter) init() {
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
	b.init()

	return io.Copy(b.sizedDigestWriter, r)
}

func (b *blobWriter) Write(p []byte) (int, error) {
	b.init()

	return b.sizedDigestWriter.Write(p)
}

func (b *blobWriter) StartedAt() time.Time {
	return b.startedAt
}

func (b *blobWriter) ID() string {
	return fmt.Sprintf("%d", b.startedAt.Unix())
}

func (b *blobWriter) Commit(ctx context.Context, provisional distribution.Descriptor) (distribution.Descriptor, error) {
	if err := b.Close(); err != nil {
		return distribution.Descriptor{}, err
	}

	b.wg.Wait()

	if err := b.err; err != nil {
		return distribution.Descriptor{}, err
	}

	d := distribution.Descriptor{}
	d.Size = b.Size()
	d.Digest = b.Digest()

	if provisional.Size > 0 {
		if d.Size != provisional.Size {
			return distribution.Descriptor{}, errors.Wrapf(distribution.ErrBlobInvalidLength, "expect %d, but got %d", provisional.Size, d.Size)
		}
	}

	if provisional.Digest != "" {
		if d.Digest != provisional.Digest {
			return distribution.Descriptor{}, distribution.ErrBlobInvalidDigest{
				Digest: d.Digest,
				Reason: fmt.Errorf("not match %s", provisional.Digest),
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
	d := a.b.Digest()
	return v1.Hash{Algorithm: string(d.Algorithm()), Hex: d.Hex()}, nil
}

func (a *proxyLayer) Size() (int64, error) {
	return a.b.Size(), nil
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
