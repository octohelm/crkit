package artifact

import (
	"bytes"
	"encoding/json"
	"sync/atomic"

	"github.com/octohelm/crkit/pkg/ociutil"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func WithAnnotations(annotations map[string]string) Option {
	return func(i AnnotationSetter) {
		for k, v := range annotations {
			i.SetAnnotation(k, v)
		}
	}
}

type AnnotationSetter interface {
	SetAnnotation(k string, v string)
}

type Option = func(i AnnotationSetter)

func Artifact(img v1.Image, c Config, optFns ...Option) (v1.Image, error) {
	i := &artifactImage{
		Image:  img,
		config: c,
	}

	for _, optFn := range optFns {
		optFn(i)
	}

	return ociutil.Image(i), nil
}

type artifactImage struct {
	v1.Image
	config Config
	m      atomic.Pointer[specv1.Manifest]

	annotations map[string]string
}

func (img *artifactImage) SetAnnotation(k string, v string) {
	if img.annotations == nil {
		img.annotations = map[string]string{}
	}
	img.annotations[k] = v
}

func (img *artifactImage) MediaType() (types.MediaType, error) {
	return types.OCIManifestSchema1, nil
}

func (img *artifactImage) ArtifactType() (string, error) {
	return img.config.ArtifactType()
}

func (img *artifactImage) RawConfigFile() ([]byte, error) {
	return img.config.RawConfigFile()
}

func (img *artifactImage) RawManifest() ([]byte, error) {
	m, err := img.OCIManifest()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(m, "", "  ")
}

func (img *artifactImage) OCIManifest() (*specv1.Manifest, error) {
	if m := img.m.Load(); m != nil {
		return m, nil
	}

	configRaw, err := img.RawConfigFile()
	if err != nil {
		return nil, err
	}

	cfgHash, cfgSize, err := v1.SHA256(bytes.NewReader(configRaw))
	if err != nil {
		return nil, err
	}

	mediaType, err := img.MediaType()
	if err != nil {
		return nil, err
	}

	artifactType, err := img.config.ArtifactType()
	if err != nil {
		return nil, err
	}

	configMediaType, err := img.config.ConfigMediaType()
	if err != nil {
		return nil, err
	}

	m := &specv1.Manifest{
		MediaType:    string(mediaType),
		ArtifactType: artifactType,
		Config: specv1.Descriptor{
			MediaType: configMediaType,
			Size:      cfgSize,
			Digest:    digest.Digest(cfgHash.String()),
		},
		Annotations: img.annotations,
	}
	m.SchemaVersion = 2

	layers, err := img.Image.Layers()
	if err != nil {
		return nil, err
	}

	for _, l := range layers {
		desc, err := partial.Descriptor(l)
		if err != nil {
			return nil, err
		}

		d := specv1.Descriptor{
			MediaType:    string(desc.MediaType),
			Digest:       digest.Digest(desc.Digest.String()),
			Size:         desc.Size,
			Annotations:  desc.Annotations,
			ArtifactType: desc.ArtifactType,
		}

		if p := desc.Platform; p != nil {
			d.Platform = &specv1.Platform{
				Architecture: p.Architecture,
				OS:           p.OS,
				OSVersion:    p.OSVersion,
				OSFeatures:   p.OSFeatures,
				Variant:      p.Variant,
			}
		}

		m.Layers = append(m.Layers, d)
	}

	img.m.Store(m)

	return m, nil
}
