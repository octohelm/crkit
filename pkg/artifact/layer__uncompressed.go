package artifact

import (
	"io"
	"sync"

	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type UncompressedLayer interface {
	MediaType() (types.MediaType, error)
	Uncompressed() (io.ReadCloser, error)
}

func NonCompressedLayer(u UncompressedLayer) containerregistryv1.Layer {
	return &uncompressedLayer{
		UncompressedLayer: u,
	}
}

type uncompressedLayer struct {
	UncompressedLayer

	hashSizeError error
	hash          containerregistryv1.Hash
	size          int64
	once          sync.Once
}

func (a *uncompressedLayer) DiffID() (containerregistryv1.Hash, error) {
	a.calcSizeHash()

	return a.hash, a.hashSizeError
}

func (a *uncompressedLayer) Size() (int64, error) {
	a.calcSizeHash()

	return a.size, a.hashSizeError
}

func (a *uncompressedLayer) calcSizeHash() {
	a.once.Do(func() {
		r, err := a.Uncompressed()
		if err != nil {
			a.hashSizeError = err
			return
		}
		defer r.Close()
		a.hash, a.size, a.hashSizeError = containerregistryv1.SHA256(r)
	})
}

func (a *uncompressedLayer) Digest() (containerregistryv1.Hash, error) {
	return a.DiffID()
}

func (a *uncompressedLayer) Compressed() (io.ReadCloser, error) {
	return a.Uncompressed()
}
