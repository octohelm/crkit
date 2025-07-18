package ocitar

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sync"

	"github.com/containerd/containerd/images"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

var ociLayoutRaw = []byte(`{"imageLayoutVersion":"1.0.0"}`)

func Write(w io.Writer, idx v1.ImageIndex) error {
	tw := tar.NewWriter(w)
	defer func() {
		_ = tw.Close()
	}()

	ww := &ociTarWriter{Writer: tw}

	return ww.writeRootIndex(idx)
}

type ociTarWriter struct {
	*tar.Writer

	blobs sync.Map
}

func (w *ociTarWriter) writeRootIndex(idx v1.ImageIndex) error {
	if err := w.writeDeps(idx, nil); err != nil {
		return fmt.Errorf("write deps failed: %w", err)
	}

	raw, err := idx.RawManifest()
	if err != nil {
		return err
	}

	b := &bytes.Buffer{}
	if err := json.Indent(b, raw, "", "  "); err != nil {
		return err
	}

	if err := w.writeToTar(tar.Header{
		Name: "index.json",
		Size: int64(b.Len()),
	}, b); err != nil {
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

func (w *ociTarWriter) writeToTarWithDigest(dgst v1.Hash, size int64, r io.Reader, scope *v1.Descriptor) error {
	// avoid dup blob write
	if _, ok := w.blobs.Load(dgst); ok {
		return nil
	}

	defer func() {
		w.blobs.Store(dgst, true)
	}()

	return w.writeToTar(tar.Header{
		Name: path.Join("blobs", dgst.Algorithm, dgst.Hex),
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

func (w *ociTarWriter) writeDeps(m manifest, scope *v1.Descriptor) error {
	switch x := m.(type) {
	case v1.ImageIndex:
		return w.writeChildren(x, scope)
	case v1.Image:
		return w.writeLayers(x, scope)
	}
	return nil
}

func (w *ociTarWriter) writeChildren(idx v1.ImageIndex, scope *v1.Descriptor) error {
	children, err := partial.Manifests(idx)
	if err != nil {
		return fmt.Errorf("resolve manifests failed, %T: %w", idx, err)
	}

	index, err := idx.IndexManifest()
	if err != nil {
		return fmt.Errorf("resolve index manifests failed, %T: %w", idx, err)
	}

	for i, child := range children {
		if i <= len(index.Manifests) {
			s := index.Manifests[i]
			if s.Annotations != nil {
				if _, ok := s.Annotations[images.AnnotationImageName]; ok {
					scope = &s
				}
			}
		}
		if err := w.writeChild(child, scope); err != nil {
			return err
		}
	}
	return nil
}

func (w *ociTarWriter) writeChild(child partial.Describable, scope *v1.Descriptor) error {
	switch child := child.(type) {
	case v1.ImageIndex:
		return w.writeManifest(child, scope)
	case v1.Image:
		return w.writeManifest(child, scope)
	case v1.Layer:
		return w.writeLayer(child, scope)
	default:
		// This can't happen.
		return fmt.Errorf("encountered unknown child: %T", child)
	}
}

func (w *ociTarWriter) writeLayers(img v1.Image, scope *v1.Descriptor) error {
	ls, err := img.Layers()
	if err != nil {
		return fmt.Errorf("resolve layers failed: %w", err)
	}

	for _, l := range ls {
		if err := w.writeLayer(l, scope); err != nil {
			return err
		}
	}
	cl, err := partial.ConfigLayer(img)
	if err != nil {
		return fmt.Errorf("resolve config failed: %w", err)
	}
	return w.writeLayer(cl, scope)
}

func (w *ociTarWriter) writeManifest(m manifest, scope *v1.Descriptor) error {
	if err := w.writeDeps(m, scope); err != nil {
		return err
	}

	raw, err := m.RawManifest()
	if err != nil {
		return fmt.Errorf("read raw manifest failed: %w", err)
	}

	dgst, err := m.Digest()
	if err != nil {
		return fmt.Errorf("read digest failed: %w", err)
	}

	return w.writeToTar(tar.Header{
		Name: path.Join("blobs", dgst.Algorithm, dgst.Hex),
		Size: int64(len(raw)),
	}, bytes.NewReader(raw))
}

func (w *ociTarWriter) writeLayer(layer v1.Layer, scope *v1.Descriptor) error {
	dgst, err := layer.Digest()
	if err != nil {
		return fmt.Errorf("read layer digest failed: %w", err)
	}

	size, err := layer.Size()
	if err != nil {
		return fmt.Errorf("read layer digest failed: %w", err)
	}

	r, err := layer.Compressed()
	if err != nil {
		return fmt.Errorf("read layer contents: %w", err)
	}

	defer func() {
		_ = r.Close()
	}()

	if err := w.writeToTarWithDigest(dgst, size, r, scope); err != nil {
		return fmt.Errorf("copy %s failed: %w", dgst, err)
	}

	return nil
}
