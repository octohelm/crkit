package ociutil

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

func FromRaw(img v1.Image) v1.Image {
	return &rawImage{Image: img}
}

type rawImage struct {
	v1.Image
}

type WithArtifactType interface {
	ArtifactType() (string, error)
}

func (i *rawImage) ArtifactType() (string, error) {
	if withAt, ok := i.Image.(WithArtifactType); ok {
		return withAt.ArtifactType()
	}
	return "", nil
}

func (i *rawImage) ConfigName() (v1.Hash, error) {
	return partial.ConfigName(i)
}

func (i *rawImage) ConfigFile() (*v1.ConfigFile, error) {
	return partial.ConfigFile(i)
}

func (i *rawImage) Manifest() (*v1.Manifest, error) {
	return partial.Manifest(i)
}

func (i *rawImage) Size() (int64, error) {
	return partial.Size(i)
}

func (i *rawImage) Digest() (v1.Hash, error) {
	return partial.Digest(i)
}
