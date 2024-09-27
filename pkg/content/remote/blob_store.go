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

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
	"github.com/opencontainers/go-digest"
)

var _ content.BlobStore = &blobStore{}

type blobStore struct {
	named  reference.Named
	client *Client
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

	i, _ := strconv.ParseInt(meta.Get("Content-Length"), 64, 10)

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

	return &blobWriter{
		id:        digest.FromString(location).Hex(),
		digester:  digest.SHA256.Digester(),
		blobStore: bs,
		location:  location,
	}, nil
}

var _ content.BlobWriter = &blobWriter{}

type blobWriter struct {
	*blobStore

	id       string
	location string

	digester digest.Digester
	offset   int64
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
	return b.offset
}

func (b *blobWriter) Digest(ctx context.Context) digest.Digest {
	return b.digester.Digest()
}

func (b *blobWriter) endpoint() string {
	location := b.location
	if strings.HasPrefix(location, "/") {
		location = b.client.Endpoint + location
	}
	return location
}

func (b *blobWriter) Write(p []byte) (int, error) {
	n := len(p)

	req, err := http.NewRequest("PATCH", b.endpoint(), io.NopCloser(bytes.NewBuffer(p[:n])))
	if err != nil {
		return -1, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Range", fmt.Sprintf("%d-%d", b.offset, b.offset+int64(n)))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", n))

	meta, err := b.client.Do(context.Background(), req).Into(nil)
	if err != nil {
		return -1, err
	}

	b.offset += int64(n)
	b.digester.Hash().Write(p[:n])

	b.location = meta.Get("Location")

	return n, nil
}

func (b *blobWriter) Commit(ctx context.Context, expect manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	dgst := b.Digest(ctx)

	req, err := http.NewRequest("PUT", b.endpoint(), nil)
	if err != nil {
		return nil, err
	}

	q := &url.Values{}
	q.Set("digest", cmp.Or(string(expect.Digest), string(dgst)))
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
