package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/distribution/reference"
	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/statuserror"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	contentutil "github.com/octohelm/crkit/pkg/content/util"
	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
	"github.com/octohelm/unifs/pkg/units"
	"github.com/opencontainers/go-digest"
)

type blobStore struct {
	named  reference.Named
	client courier.Client
}

var _ content.Provider = &blobStore{}

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

var _ content.Remover = &blobStore{}

func (bs *blobStore) Remove(ctx context.Context, dgst digest.Digest) error {
	req := &registry.DeleteBlob{}
	req.Name = content.Name(bs.named.Name())
	req.Digest = content.Digest(dgst)

	_, _, err := Do(ctx, bs.client, req)
	return err
}

func (bs *blobStore) Resume(ctx context.Context, id string) (content.BlobWriter, error) {
	bw := &blobWriter{
		ctx:       ctx,
		blobStore: bs,
		chunk:     bytes.NewBuffer(nil),
	}
	bw.location = fmt.Sprintf("/v2/%s/blobs/uploads/%s", bw.named.Name(), id)
	bw.id = id
	return bw, nil
}

func (bs *blobStore) Writer(ctx context.Context) (content.BlobWriter, error) {
	req := &registry.UploadBlob{}
	req.Name = content.Name(bs.named.Name())

	_, meta, err := Do(ctx, bs.client, req)
	if err != nil {
		return nil, err
	}

	bw := &blobWriter{
		ctx:       ctx,
		blobStore: bs,
		chunk:     bytes.NewBuffer(nil),
	}

	if err := bw.syncFromMeta(meta); err != nil {
		return nil, err
	}

	return bw, nil
}

var _ content.BlobWriter = &blobWriter{}

type blobWriter struct {
	ctx context.Context

	*blobStore
	chunk *bytes.Buffer

	id       string
	location string
	written  int64

	err  error
	once sync.Once

	chunkMinLength int64
	digest         digest.Digest
}

func (bw *blobWriter) ID() string {
	return bw.id
}

func (bw *blobWriter) Size(ctx context.Context) int64 {
	return bw.written
}

func (bw *blobWriter) Digest(ctx context.Context) digest.Digest {
	return bw.digest
}

func (bw *blobWriter) endpoint() string {
	location := bw.location

	if strings.HasPrefix(location, "/") {
		if endpoint, ok := bw.client.(interface{ GetEndpoint() string }); ok {
			location = endpoint.GetEndpoint() + location
		}
	}

	return location
}

func (bw *blobWriter) Write(p []byte) (int, error) {
	n := len(p)

	bw.chunk.Write(p[:n])

	if int64(bw.chunk.Len()) >= bw.chunkMinLength {
		if err := bw.sendChunkIfNeed(bw.ctx); err != nil {
			return -1, err
		}
	}

	return n, nil
}

func (bw *blobWriter) Cancel(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, bw.endpoint(), nil)
	if err != nil {
		return err
	}
	if _, err := bw.client.Do(context.Background(), req).Into(nil); err != nil {
		return err
	}
	return nil
}

func (bw *blobWriter) Close() error {
	bw.once.Do(func() {
		bw.err = bw.sendChunkIfNeed(bw.ctx)
	})

	return bw.err
}

func (bw *blobWriter) Commit(ctx context.Context, expect manifestv1.Descriptor) (*manifestv1.Descriptor, error) {
	var req *http.Request

	if n := int64(bw.chunk.Len()); n == 0 {
		r, err := http.NewRequestWithContext(ctx, http.MethodPut, bw.endpoint(), nil)
		if err != nil {
			return nil, err
		}
		req = r
	} else {
		r, err := http.NewRequestWithContext(ctx, http.MethodPut, bw.endpoint(), io.NopCloser(bw.chunk))
		if err != nil {
			return nil, err
		}
		bw.patchRequestContentLength(r, n)
		req = r
	}

	q := &url.Values{}
	if expect.Digest != "" {
		q.Set("digest", string(expect.Digest))
	}

	req.URL.RawQuery = q.Encode()

	meta, err := bw.client.Do(ctx, req).Into(nil)
	if err != nil {
		return nil, err
	}

	d := &manifestv1.Descriptor{}

	d.Digest = digest.Digest(meta.Get("Docker-Content-Digest"))
	d.Size = bw.Size(ctx)

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

func (bw *blobWriter) patchRequestContentLength(req *http.Request, n int64) {
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Range", (&contentutil.Range{Start: bw.written, Length: n}).String())
	req.ContentLength = n
}

func (bw *blobWriter) sendChunkIfNeed(ctx context.Context) error {
	if bw.chunk.Len() == 0 {
		return nil
	}
	return bw.sendChunk(ctx, false)
}

func (bw *blobWriter) sendChunk(ctx context.Context, retry bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, bw.endpoint(), io.NopCloser(bw.chunk))
	if err != nil {
		return err
	}

	n := int64(bw.chunk.Len())

	bw.patchRequestContentLength(req, n)

	meta, err := bw.client.Do(ctx, req).Into(nil)
	if err != nil {
		if !retry {
			d := &statuserror.Descriptor{}
			if errors.As(err, &d) {
				if d.Status == http.StatusRequestedRangeNotSatisfiable {
					if err := bw.syncFromBlobUpload(ctx); err != nil {
						return err
					}
					return bw.sendChunk(ctx, true)
				}
			}
		}
		return err
	}

	bw.chunk.Reset()
	bw.written += n

	return bw.syncFromMeta(meta)
}

func (bw *blobWriter) syncFromBlobUpload(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bw.endpoint(), nil)
	if err != nil {
		return err
	}

	meta, err := bw.client.Do(ctx, req).Into(nil)
	if err != nil {
		return err
	}

	return bw.syncFromMeta(meta)
}

func (bw *blobWriter) syncFromMeta(meta courier.Metadata) error {
	if chunkMinLength := meta.Get("OCI-Chunk-Min-Length"); chunkMinLength != "" {
		i, _ := strconv.ParseInt(chunkMinLength, 10, 64)
		if i > 0 {
			bw.chunkMinLength = i
		}
	}

	if bw.chunkMinLength == 0 {
		bw.chunkMinLength = int64(20 * units.MiB)
	}

	if location := meta.Get("Location"); location != "" {
		bw.location = location

		fmt.Println("Location", location)

		if bw.id == "" {
			parts := strings.SplitN(bw.location, "/blobs/uploads/", 2)
			if len(parts) == 2 {
				bw.id = strings.Trim(parts[1], "/")
			}
		}
	}

	if rng := meta.Get("Range"); rng != "" {
		r, err := contentutil.ParseRange(rng)
		if err != nil {
			return err
		}
		bw.written = r.Length
	}

	return nil
}
