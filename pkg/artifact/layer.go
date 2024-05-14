package artifact

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type Layer = v1.Layer

func FromBytes(mediaType string, data []byte) (Layer, error) {
	return FromOpener(mediaType, func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(data)), nil
	})
}

func FromReader(mediaType string, r io.Reader) (Layer, error) {
	return FromOpener(mediaType, func() (io.ReadCloser, error) {
		return io.NopCloser(r), nil
	})
}

func FromOpener(mediaType string, uncompressed func() (io.ReadCloser, error)) (Layer, error) {
	return NonCompressedLayer(&artifact{
		mediaType:    mediaType,
		uncompressed: uncompressed,
	}), nil
}

type artifact struct {
	mediaType    string
	uncompressed func() (io.ReadCloser, error)
}

func (a *artifact) MediaType() (types.MediaType, error) {
	return types.MediaType(a.mediaType), nil
}

func (a *artifact) Uncompressed() (io.ReadCloser, error) {
	return a.uncompressed()
}

func Gzipped(l Layer) Layer {
	return &compressed{
		Layer: l,
		compressedLayer: sync.OnceValues(func() (v1.Layer, error) {
			return partial.UncompressedToLayer(l)
		}),
	}
}

type compressed struct {
	Layer
	compressedLayer func() (v1.Layer, error)
}

func (a *compressed) MediaType() (types.MediaType, error) {
	m, err := a.Layer.MediaType()
	if err != nil {
		return "", err
	}
	return types.MediaType(fmt.Sprintf("%s+gzip", m)), nil
}

func (a *compressed) Compressed() (io.ReadCloser, error) {
	l, err := a.compressedLayer()
	if err != nil {
		return nil, err
	}
	return l.Compressed()
}

func (a *compressed) Digest() (v1.Hash, error) {
	l, err := a.compressedLayer()
	if err != nil {
		return v1.Hash{}, err
	}
	return l.Digest()
}

func WithDescriptor(l Layer, descriptor v1.Descriptor) Layer {
	return &artifactWithDescriptor{
		desc:  descriptor,
		Layer: l,
	}
}

type artifactWithDescriptor struct {
	Layer

	desc v1.Descriptor
}

func (w *artifactWithDescriptor) Descriptor() (*v1.Descriptor, error) {
	return &w.desc, nil
}
