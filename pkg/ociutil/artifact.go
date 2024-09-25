package ociutil

import (
	"encoding/json"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

type WithArtifactType interface {
	ArtifactType() (string, error)
}

func Artifact(img v1.Image, artifactType string) v1.Image {
	return &artifact{
		artifactType: artifactType,
		Image:        img,
	}
}

type artifact struct {
	artifactType string
	v1.Image
}

func (i *artifact) ArtifactType() (string, error) {
	return i.artifactType, nil
}

func (i *artifact) ConfigName() (v1.Hash, error) {
	return partial.ConfigName(i)
}

func (i *artifact) ConfigFile() (*v1.ConfigFile, error) {
	return partial.ConfigFile(i)
}

func (i *artifact) Manifest() (*v1.Manifest, error) {
	return partial.Manifest(i)
}

func (i *artifact) Size() (int64, error) {
	return partial.Size(i)
}

func (i *artifact) Digest() (v1.Hash, error) {
	return partial.Digest(i)
}

func ArtifactType(img partial.WithRawManifest) (string, error) {
	if p, ok := img.(WithArtifactType); ok {
		return p.ArtifactType()
	}

	raw, err := img.RawManifest()
	if err != nil {
		return "", err
	}

	m := &struct {
		ArtifactType string `json:"artifactType,omitempty"`
	}{}

	if err := json.Unmarshal(raw, m); err != nil {
		return "", err
	}

	return m.ArtifactType, nil
}
