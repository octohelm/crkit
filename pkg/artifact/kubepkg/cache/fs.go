package cache

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/opencontainers/go-digest"
)

type fscache struct {
	path string
}

func NewFilesystemCache(path string) cache.Cache {
	return &fscache{path}
}

func (fs *fscache) Put(l v1.Layer) (v1.Layer, error) {
	dgst, err := l.Digest()
	if err != nil {
		return nil, err
	}
	diffID, err := l.DiffID()
	if err != nil {
		return nil, err
	}
	return &layer{
		Layer:  l,
		path:   fs.path,
		digest: dgst,
		diffID: diffID,
	}, nil
}

type layer struct {
	v1.Layer
	path           string
	digest, diffID v1.Hash
}

func (l *layer) create(h v1.Hash) (io.WriteCloser, error) {
	tmpPath := injestpath(l.path, h)

	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o700); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(tmpPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	d := digest.SHA256.Digester()

	return &writer{
		tmpPath:   tmpPath,
		cachePath: cachepath(l.path, h),
		expect:    h,
		f:         f,
		digester:  d,
		Writer:    io.MultiWriter(f, d.Hash()),
	}, nil
}

type writer struct {
	tmpPath   string
	cachePath string

	expect   v1.Hash
	f        *os.File
	digester digest.Digester

	io.Writer
}

func (w *writer) Close() error {
	if err := w.f.Close(); err != nil {
		return err
	}

	d := w.digester.Digest()
	if d.String() != w.expect.String() {
		return errors.New("digest not match")
	}

	if err := os.MkdirAll(filepath.Dir(w.cachePath), 0o700); err != nil {
		return err
	}

	return os.Rename(w.tmpPath, w.cachePath)
}

func (l *layer) Compressed() (io.ReadCloser, error) {
	f, err := l.create(l.digest)
	if err != nil {
		return nil, err
	}
	rc, err := l.Layer.Compressed()
	if err != nil {
		return nil, err
	}
	return &readcloser{
		t:      io.TeeReader(rc, f),
		closes: []func() error{rc.Close, f.Close},
	}, nil
}

func (l *layer) Uncompressed() (io.ReadCloser, error) {
	f, err := l.create(l.diffID)
	if err != nil {
		return nil, err
	}
	rc, err := l.Layer.Uncompressed()
	if err != nil {
		return nil, err
	}
	return &readcloser{
		t:      io.TeeReader(rc, f),
		closes: []func() error{rc.Close, f.Close},
	}, nil
}

type readcloser struct {
	t      io.Reader
	closes []func() error
}

func (rc *readcloser) Read(b []byte) (int, error) {
	return rc.t.Read(b)
}

func (rc *readcloser) Close() error {
	// Call all Close methods, even if any returned an error. Return the
	// first returned error.
	var err error
	for _, c := range rc.closes {
		lastErr := c()
		if err == nil {
			err = lastErr
		}
	}
	return err
}

func (fs *fscache) Get(h v1.Hash) (v1.Layer, error) {
	l, err := tarball.LayerFromFile(cachepath(fs.path, h))
	if os.IsNotExist(err) {
		return nil, cache.ErrNotFound
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		// Delete and return ErrNotFound because the layer was incomplete.
		if err := fs.Delete(h); err != nil {
			return nil, err
		}
		return nil, cache.ErrNotFound
	}
	return l, err
}

func (fs *fscache) Delete(h v1.Hash) error {
	err := os.Remove(cachepath(fs.path, h))
	if os.IsNotExist(err) {
		return cache.ErrNotFound
	}
	return err
}

func cachepath(path string, h v1.Hash) string {
	return filepath.Join(path, "blobs", h.Algorithm, h.Hex)
}

func injestpath(path string, h v1.Hash) string {
	return filepath.Join(path, "injests", h.Hex)
}
