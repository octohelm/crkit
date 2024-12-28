package remote

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alecthomas/units"
	"github.com/distribution/reference"
	"github.com/octohelm/courier/pkg/courier"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	contentutil "github.com/octohelm/crkit/pkg/content/util"
	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
	"github.com/opencontainers/go-digest"
)

var _ content.BlobStore = &blobStore{}

type blobStore struct {
	named  reference.Named
	client courier.Client
}

func (bs *blobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	req := &registry.DeleteBlob{}
	req.Name = content.Name(bs.named.Name())
	req.Digest = content.Digest(dgst)

	_, _, err := Do(ctx, bs.client, req)
	return err
}

func (bs *blobStore) Info(ctx context.Context, dgst digest.Digest) (*manifestv1.Descriptor, error) {
	req := &registry.HeadBlob{}
	req.Name = content.Name(bs.named.Name())
	req.Digest = content.Digest(dgst)

	_, meta, err := Do(ctx, bs.client, req)
	if err != nil {
		return nil, err
	}

	i, err := strconv.ParseInt(meta.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("HEAD response header missing Content-Length: %w", err)
	}

	return &manifestv1.Descriptor{
		MediaType: meta.Get("Content-Type"),
		Digest:    digest.Digest(meta.Get("Docker-Content-Digest")),
		Size:      i,
	}, nil
}

func (bs *blobStore) Open(ctx context.Context, dgst digest.Digest) (r io.ReadCloser, err error) {
	req := &registry.GetBlob{}
	req.Name = content.Name(bs.named.Name())
	req.Digest = content.Digest(dgst)

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		if _, err := bs.client.Do(ctx, req).Into(pw); err != nil {
			return
		}
	}()

	return pr, nil
}

func (bs *blobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	req := &registry.UploadBlob{}
	req.Name = content.Name(bs.named.Name())

	_, meta, err := Do(ctx, bs.client, req)
	if err != nil {
		return nil, err
	}

	location := meta.Get("Location")

	bw := &blobWriter{
		id:        digest.FromString(location).Hex(),
		digester:  digest.SHA256.Digester(),
		blobStore: bs,
		location:  location,

		chunkMinLength: int64(20 * units.MiB),
		chunk:          bytes.NewBuffer(nil),
	}

	chunkMinLength := meta.Get("OCI-Chunk-Min-Length")
	if chunkMinLength != "" {
		i, _ := strconv.ParseInt(chunkMinLength, 10, 64)
		if i > 0 {
			bw.chunkMinLength = i
		}
	}

	return bw, nil
}

var _ content.BlobWriter = &blobWriter{}

type blobWriter struct {
	*blobStore

	id       string
	location string

	chunkMinLength int64
	chunk          *bytes.Buffer
	chunkOffset    int64

	digester digest.Digester
}

func (b *blobWriter) ID() string {
	return b.id
}

func (b *blobWriter) Close() error {
	req := &registry.CancelUploadBlob{}
	req.Name = content.Name(b.named.Name())
	req.ID = b.id
	return nil
}

func (b *blobWriter) Size(ctx context.Context) int64 {
	return b.chunkOffset
}

func (b *blobWriter) Digest(ctx context.Context) digest.Digest {
	return b.digester.Digest()
}

func (b *blobWriter) endpoint() string {
	location := b.location
	if strings.HasPrefix(location, "/") {
		if endpoint, ok := b.client.(interface{ GetEndpoint() string }); ok {
			location = endpoint.GetEndpoint() + location
		}
	}
	return location
}

func (b *blobWriter) Write(p []byte) (int, error) {
	n := len(p)

	b.chunk.Write(p[:n])
	b.digester.Hash().Write(p[:n])

	if int64(b.chunk.Len()) >= b.chunkMinLength {
		if err := b.sendChunk(); err != nil {
			return -1, err
		}
	}

	return n, nil
}

func (b *blobWriter) sendChunk() error {
	req, err := http.NewRequest("PATCH", b.endpoint(), io.NopCloser(b.chunk))
	if err != nil {
		return err
	}

	n := int64(b.chunk.Len())

	b.patchContentLength(req, n)

	meta, err := b.client.Do(context.Background(), req).Into(nil)
	if err != nil {
		return err
	}

	b.chunk.Reset()
	b.location = meta.Get("Location")
	b.chunkOffset += n

	return nil
}

func (b *blobWriter) patchContentLength(req *http.Request, n int64) {
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Range", (contentutil.Range{Start: b.chunkOffset, Length: n}).String())
	req.ContentLength = n
}

func (b *blobWriter) Commit(ctx context.Context, expect manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	dgst := b.Digest(ctx)

	var req *http.Request

	if n := int64(b.chunk.Len()); n == 0 {
		r, err := http.NewRequest("PUT", b.endpoint(), nil)
		if err != nil {
			return nil, err
		}
		req = r
	} else {
		r, err := http.NewRequest("PUT", b.endpoint(), io.NopCloser(b.chunk))
		if err != nil {
			return nil, err
		}

		b.patchContentLength(r, n)
		req = r
	}

	q := &url.Values{}
	if expect.Digest != "" {
		q.Set("digest", cmp.Or(string(expect.Digest), string(dgst)))
	}

	req.URL.RawQuery = q.Encode()

	if _, err := b.client.Do(ctx, req).Into(nil); err != nil {
		return nil, err
	}

	d := &manifestv1.Descriptor{}
	d.Digest = dgst
	d.Size = b.Size(ctx)

	if expect.Size > 0 {
		if d.Size != expect.Size {
			return nil, &content.ErrBlobInvalidLength{
				Reason: fmt.Sprintf("expect %d, but got %d", expect.Size, d.Size),
			}
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
