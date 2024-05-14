package ocitar

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/octohelm/x/ptr"

	"github.com/pkg/errors"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

var ociLayoutRaw = []byte(`{"imageLayoutVersion":"1.0.0"}`)

func Write(w io.Writer, idx v1.ImageIndex, options ...Option) error {
	tw := tar.NewWriter(w)
	defer func() {
		_ = tw.Close()
	}()

	ww := &ociTarWriter{Writer: tw}
	ww.Build(options...)

	return ww.writeRootIndex(idx)
}

type ociTarWriter struct {
	*tar.Writer

	blobs sync.Map

	progress *progress
}

func (w *ociTarWriter) Build(options ...Option) {
	for _, o := range options {
		o(w)
	}
}

type Option = func(*ociTarWriter)

func WithProgress(updates chan<- Update) Option {
	return func(w *ociTarWriter) {
		w.progress = &progress{updates: updates}
	}
}

func (w *ociTarWriter) writeRootIndex(idx v1.ImageIndex) error {
	if err := w.writeDeps(idx); err != nil {
		return errors.Wrap(err, "write deps failed")
	}

	raw, err := idx.RawManifest()
	if err != nil {
		return err
	}

	if err := w.writeToTar(tar.Header{
		Name: "index.json",
		Size: int64(len(raw)),
	}, bytes.NewBuffer(raw)); err != nil {
		return err
	}

	if err := w.writeToTar(tar.Header{
		Name: "oci-layout",
		Size: int64(len(ociLayoutRaw)),
	}, bytes.NewBuffer(ociLayoutRaw)); err != nil {
		return err
	}

	return nil
}

type manifest interface {
	partial.Describable

	RawManifest() ([]byte, error)
}

func (w *ociTarWriter) writeToTarWithDigest(dgst v1.Hash, size int64, r io.Reader) error {
	// avoid dup blob write
	if _, ok := w.blobs.Load(dgst); ok {
		return nil
	}

	defer func() {
		w.blobs.Store(dgst, true)
	}()

	if w.progress != nil {
		r = &progressReader{
			r:        r,
			digest:   dgst,
			total:    size,
			count:    ptr.Ptr(int64(0)),
			progress: w.progress,
		}
	}

	return w.writeToTar(tar.Header{
		Name: filepath.Join("blobs", dgst.Algorithm, dgst.Hex),
		Size: size,
	}, r)
}

func (w *ociTarWriter) writeToTar(header tar.Header, r io.Reader) error {
	header.Mode = 0o644
	if err := w.WriteHeader(&header); err != nil {
		return err
	}

	if _, err := io.CopyN(w, r, header.Size); err != nil {
		return err
	}
	return nil
}

func (w *ociTarWriter) writeDeps(m manifest) error {
	switch x := m.(type) {
	case v1.ImageIndex:
		return w.writeChildren(x)
	case v1.Image:
		return w.writeLayers(x)
	}
	return nil
}

func (w *ociTarWriter) writeChildren(idx v1.ImageIndex) error {
	children, err := partial.Manifests(idx)
	if err != nil {
		return errors.Wrapf(err, "resolve manifests failed, %T", idx)
	}
	for _, child := range children {
		if err := w.writeChild(child); err != nil {
			return err
		}
	}
	return nil
}

func (w *ociTarWriter) writeChild(child partial.Describable) error {
	switch child := child.(type) {
	case v1.ImageIndex:
		return w.writeManifest(child)
	case v1.Image:
		return w.writeManifest(child)
	case v1.Layer:
		return w.writeLayer(child)
	default:
		// This can't happen.
		return fmt.Errorf("encountered unknown child: %T", child)
	}
}

func (w *ociTarWriter) writeLayers(img v1.Image) error {
	ls, err := img.Layers()
	if err != nil {
		return errors.Wrap(err, "resolve layers failed")
	}

	for _, l := range ls {
		if err := w.writeLayer(l); err != nil {
			return err
		}
	}
	cl, err := partial.ConfigLayer(img)
	if err != nil {
		return errors.Wrap(err, "resolve config failed")
	}
	return w.writeLayer(cl)
}

func (w *ociTarWriter) writeManifest(m manifest) error {
	if err := w.writeDeps(m); err != nil {
		return err
	}

	raw, err := m.RawManifest()
	if err != nil {
		return errors.Wrap(err, "read raw manifest failed")
	}

	dgst, err := m.Digest()
	if err != nil {
		return errors.Wrap(err, "read digest failed")
	}

	return w.writeToTar(tar.Header{
		Name: filepath.Join("blobs", dgst.Algorithm, dgst.Hex),
		Size: int64(len(raw)),
	}, bytes.NewReader(raw))
}

func (w *ociTarWriter) writeLayer(layer v1.Layer) error {
	dgst, err := layer.Digest()
	if err != nil {
		return errors.Wrap(err, "read layer digest failed")
	}

	size, err := layer.Size()
	if err != nil {
		return errors.Wrap(err, "read layer digest failed")
	}

	r, err := layer.Compressed()
	if err != nil {
		return errors.Wrap(err, "read layer contents")
	}

	defer func() {
		_ = r.Close()
	}()

	if err := w.writeToTarWithDigest(dgst, size, r); err != nil {
		return errors.Wrap(err, "copy layer failed")
	}

	return nil
}
