package artifact

import (
	"fmt"
	"io"
	"sync"

	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func Gzipped(l Layer) (Layer, error) {
	return &compressedLayerWithoutDiff{Layer: l}, nil
}

type compressedLayerWithoutDiff struct {
	Layer

	once          sync.Once
	hash          containerregistryv1.Hash
	size          int64
	hashSizeError error
}

func (a *compressedLayerWithoutDiff) MediaType() (types.MediaType, error) {
	m, err := a.Layer.MediaType()
	if err != nil {
		return "", err
	}
	return types.MediaType(fmt.Sprintf("%s+gzip", m)), nil
}

func (ule *compressedLayerWithoutDiff) Compressed() (io.ReadCloser, error) {
	u, err := partial.UncompressedToLayer(ule.Layer)
	if err != nil {
		return nil, err
	}
	return u.Compressed()
}

func (ule *compressedLayerWithoutDiff) Digest() (containerregistryv1.Hash, error) {
	ule.calcSizeHash()
	return ule.hash, ule.hashSizeError
}

// Size implements v1.Layer
func (ule *compressedLayerWithoutDiff) Size() (int64, error) {
	ule.calcSizeHash()
	return ule.size, ule.hashSizeError
}

func (ule *compressedLayerWithoutDiff) calcSizeHash() {
	ule.once.Do(func() {
		var r io.ReadCloser
		r, ule.hashSizeError = ule.Compressed()
		if ule.hashSizeError != nil {
			return
		}
		defer r.Close()
		ule.hash, ule.size, ule.hashSizeError = containerregistryv1.SHA256(r)
	})
}
