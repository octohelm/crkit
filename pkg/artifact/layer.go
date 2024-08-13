package artifact

import (
	"bytes"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"io"
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

func Gzipped(l Layer) (Layer, error) {
	gziped, err := partial.UncompressedToLayer(l)
	if err != nil {
		return gziped, nil
	}
	return &compressed{Layer: l}, nil
}

type compressed struct {
	Layer
}

func (a *compressed) MediaType() (types.MediaType, error) {
	m, err := a.Layer.MediaType()
	if err != nil {
		return "", err
	}
	return types.MediaType(fmt.Sprintf("%s+gzip", m)), nil
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
