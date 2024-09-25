package ocitar

import (
	"encoding/json"
	"io"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func Index(opener Opener) (v1.ImageIndex, error) {
	tr := &tarReader{opener: opener}

	r, err := tr.Open("index.json")
	if err != nil {
		return nil, err
	}
	return openAsIndexReader(tr, r)
}

type index struct {
	o             FileOpener
	d             v1.Descriptor
	indexManifest *v1.IndexManifest
	raw           []byte
}

type readCloser struct {
	io.Reader
	close func() error
}

func (r *readCloser) Close() error {
	return r.close()
}

func (i *index) MediaType() (types.MediaType, error) {
	return types.OCIImageIndex, nil
}

func (i *index) Digest() (v1.Hash, error) {
	return i.d.Digest, nil
}

func (i *index) Size() (int64, error) {
	return i.d.Size, nil
}

func (i *index) IndexManifest() (*v1.IndexManifest, error) {
	return i.indexManifest, nil
}

func (i *index) RawManifest() ([]byte, error) {
	return i.raw, nil
}

func (i *index) Image(h v1.Hash) (v1.Image, error) {
	for _, d := range i.indexManifest.Manifests {
		if d.MediaType.IsImage() && d.Digest == h {
			f, err := i.o.Open(layoutBlobsPath(h))
			if err != nil {
				return nil, err
			}
			return openAsImageReader(i.o, f)
		}
	}
	return nil, &ErrNotFound{Digest: h}
}

func (i *index) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	for _, d := range i.indexManifest.Manifests {
		if d.MediaType.IsIndex() && d.Digest == h {
			f, err := i.o.Open(layoutBlobsPath(h))
			if err != nil {
				return nil, err
			}
			return openAsIndexReader(i.o, f)
		}
	}
	return nil, &ErrNotFound{
		Digest: h,
	}
}

func openAsIndexReader(o FileOpener, r io.ReadCloser) (v1.ImageIndex, error) {
	defer r.Close()

	i := &index{o: o}

	digester := digest.SHA256.Digester()
	data, err := io.ReadAll(io.TeeReader(r, digester.Hash()))
	if err != nil {
		return nil, err
	}

	dgst := digester.Digest()

	i.raw = data
	i.d.Size = int64(len(data))
	i.d.Digest = v1.Hash{
		Algorithm: string(dgst.Algorithm()),
		Hex:       dgst.Hex(),
	}

	indexManifest := &v1.IndexManifest{}
	if err := json.Unmarshal(i.raw, indexManifest); err != nil {
		return nil, err
	}
	i.indexManifest = indexManifest

	return i, nil
}

func openAsImageReader(o FileOpener, r io.ReadCloser) (v1.Image, error) {
	defer r.Close()

	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	m := &specv1.Manifest{}
	if err := json.Unmarshal(raw, m); err != nil {
		return nil, err
	}

	configName := v1.Hash{
		Algorithm: string(m.Config.Digest.Algorithm()),
		Hex:       m.Config.Digest.Hex(),
	}

	configReader, err := o.Open(layoutBlobsPath(configName))
	if err != nil {
		return nil, err
	}
	defer configReader.Close()

	configRaw, err := io.ReadAll(configReader)
	if err != nil {
		return nil, err
	}

	return partial.CompressedToImage(&image{
		o:         o,
		m:         m,
		raw:       raw,
		configRaw: configRaw,
	})
}

type image struct {
	o         FileOpener
	m         *specv1.Manifest
	raw       []byte
	configRaw []byte
}

func (i *image) MediaType() (types.MediaType, error) {
	return types.MediaType(i.m.MediaType), nil
}

func (i *image) RawConfigFile() ([]byte, error) {
	return i.configRaw, nil
}

func (i *image) RawManifest() ([]byte, error) {
	return i.raw, nil
}

func (i *image) LayerByDigest(hash v1.Hash) (partial.CompressedLayer, error) {
	for _, l := range i.m.Layers {
		if l.Digest.String() == hash.String() {
			return &layer{
				d: v1.Descriptor{
					MediaType:    types.MediaType(l.MediaType),
					ArtifactType: l.ArtifactType,
					Digest:       hash,
					Size:         l.Size,
					Annotations:  l.Annotations,
				},
				opener: func() (io.ReadCloser, error) {
					return i.o.Open(layoutBlobsPath(hash))
				},
			}, nil
		}
	}
	return nil, &ErrNotFound{
		Digest: hash,
	}
}

type layer struct {
	d      v1.Descriptor
	opener Opener
}

func (l *layer) MediaType() (types.MediaType, error) {
	return l.d.MediaType, nil
}

func (l *layer) Size() (int64, error) {
	return l.d.Size, nil
}

func (l *layer) Digest() (v1.Hash, error) {
	return l.d.Digest, nil
}

func (l *layer) Compressed() (io.ReadCloser, error) {
	return l.opener()
}

func layoutBlobsPath(hash v1.Hash) string {
	return filepath.Join("blobs", hash.Algorithm, hash.Hex)
}
