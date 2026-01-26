package partial

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"sync"

	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
)

func BlobFromBytes(data []byte, descriptors ...ocispecv1.Descriptor) oci.Blob {
	return &blob{
		d: MergeDescriptors(descriptors...),
		open: func(ctx context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBuffer(data)), nil
		},
	}
}

func BlobFromOpener(open func(ctx context.Context) (io.ReadCloser, error), descriptors ...ocispecv1.Descriptor) oci.Blob {
	return &blob{
		d:    MergeDescriptors(descriptors...),
		open: open,
	}
}

type blob struct {
	d ocispecv1.Descriptor

	raw  []byte
	err  error
	once sync.Once

	open func(ctx context.Context) (io.ReadCloser, error)
}

func (l *blob) initOnce(ctx context.Context) {
	l.once.Do(func() {
		if l.open != nil {
			rc, err := l.open(ctx)
			if err != nil {
				l.err = err
				return
			}
			defer rc.Close()

			d := digest.SHA256.Digester()

			n, err := io.Copy(d.Hash(), rc)
			if err != nil {
				l.err = err
				return
			}

			l.d.Digest = d.Digest()
			l.d.Size = n
		}
	})
}

func (l *blob) Descriptor(ctx context.Context) (ocispecv1.Descriptor, error) {
	l.initOnce(ctx)

	return l.d, l.err
}

func (l *blob) Open(ctx context.Context) (io.ReadCloser, error) {
	return l.open(ctx)
}

func CompressedBlobFromOpener(d ocispecv1.Descriptor, open func(ctx context.Context) (io.ReadCloser, error)) oci.Blob {
	d.MediaType = d.MediaType + "+gzip"

	return &blob{
		d: d,
		open: func(ctx context.Context) (io.ReadCloser, error) {
			r, err := open(ctx)
			if err != nil {
				return nil, err
			}

			pr, pw := io.Pipe()

			go func() {
				defer r.Close()

				gw := gzip.NewWriter(pw)

				if _, err := io.Copy(gw, r); err != nil {
					_ = gw.Close()
					_ = pw.CloseWithError(err)
					return
				}

				_ = gw.Close()
				_ = pw.Close()
			}()

			return pr, nil
		},
	}
}
