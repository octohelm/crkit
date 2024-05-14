package artifact

import (
	"io"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type UncompressedLayer interface {
	MediaType() (types.MediaType, error)

	Uncompressed() (io.ReadCloser, error)
}

func NonCompressedLayer(u UncompressedLayer) v1.Layer {
	return &nonCompressedLayer{
		UncompressedLayer: u,
	}
}

type nonCompressedLayer struct {
	UncompressedLayer

	hashSizeError error
	hash          v1.Hash
	size          int64
	once          sync.Once
}

func (a *nonCompressedLayer) DiffID() (v1.Hash, error) {
	a.calcSizeHash()

	return a.hash, a.hashSizeError
}

func (a *nonCompressedLayer) Size() (int64, error) {
	a.calcSizeHash()

	return a.size, a.hashSizeError
}

func (a *nonCompressedLayer) Digest() (v1.Hash, error) {
	return a.DiffID()
}

func (a *nonCompressedLayer) Compressed() (io.ReadCloser, error) {
	return a.Uncompressed()
}

func (a *nonCompressedLayer) calcSizeHash() {
	a.once.Do(func() {
		r, err := a.Uncompressed()
		if err != nil {
			a.hashSizeError = err
			return
		}
		defer r.Close()
		a.hash, a.size, a.hashSizeError = v1.SHA256(r)
	})
}
