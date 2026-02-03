package internal

import (
	"bytes"
	"context"
	"io"
	"iter"
	"maps"

	"github.com/go-json-experiment/json"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
)

type ImageCore interface {
	Config(ctx context.Context) (oci.Blob, error)
	Layers(ctx context.Context) iter.Seq2[oci.Blob, error]
}

type Image struct {
	img  *ocispecv1.Manifest
	desc *ocispecv1.Descriptor
	raw  []byte
}

func (i *Image) Descriptor(ctx context.Context) (ocispecv1.Descriptor, error) {
	return *i.desc, nil
}

func (i *Image) Raw(ctx context.Context) ([]byte, error) {
	return i.raw, nil
}

func (i *Image) Value(ctx context.Context) (ocispecv1.Manifest, error) {
	return *i.img, nil
}

func (i *Image) InitFromReader(r io.Reader, descs ...ocispecv1.Descriptor) error {
	digester := digest.SHA256.Digester()
	buf := bytes.NewBuffer(nil)

	img := &ocispecv1.Manifest{}
	if err := json.UnmarshalRead(io.TeeReader(r, io.MultiWriter(buf, digester.Hash())), img); err != nil {
		return err
	}

	if len(descs) > 0 {
		i.desc = MergeDescriptors(descs...)
	} else {
		i.desc = &ocispecv1.Descriptor{
			MediaType:    img.MediaType,
			ArtifactType: img.ArtifactType,
		}
	}

	if dgst := digester.Digest(); dgst != i.desc.Digest {
		i.desc.Digest = dgst
	}

	i.desc.Size = int64(buf.Len())

	if i.desc.ArtifactType == "" {
		i.desc.ArtifactType = img.ArtifactType
	}

	if i.desc.MediaType == "" {
		i.desc.MediaType = img.MediaType
	}

	if i.desc.Platform == nil {
		i.desc.Platform = img.Config.Platform
	}

	if len(img.Annotations) > 0 {
		if i.desc.Annotations == nil {
			i.desc.Annotations = make(map[string]string)
		}
		maps.Copy(i.desc.Annotations, img.Annotations)
	}

	i.img = img
	i.raw = buf.Bytes()

	return nil
}

func (i *Image) Build(mutates ...func(m *ocispecv1.Manifest) error) error {
	img := &ocispecv1.Manifest{}
	i.img = img

	img.SchemaVersion = 2
	img.MediaType = ocispecv1.MediaTypeImageManifest

	for _, mut := range mutates {
		if err := mut(img); err != nil {
			return err
		}
	}

	digester := digest.SHA256.Digester()
	buf := bytes.NewBuffer(nil)

	if err := json.MarshalWrite(
		io.MultiWriter(buf, digester.Hash()),
		img,
		json.Deterministic(true),
	); err != nil {
		return err
	}

	d := &ocispecv1.Descriptor{
		MediaType:    img.MediaType,
		ArtifactType: img.ArtifactType,
		Digest:       digester.Digest(),
		Size:         int64(buf.Len()),
	}

	if p := img.Config.Platform; p != nil {
		d.Platform = p
	}

	if len(img.Annotations) > 0 {
		d.Annotations = make(map[string]string)
		maps.Copy(d.Annotations, img.Annotations)
	}

	i.desc = d
	i.raw = buf.Bytes()

	return nil
}

func (i *Image) CollectTo(ctx context.Context, core ImageCore, m *ocispecv1.Manifest) error {
	c, err := core.Config(ctx)
	if err != nil {
		return err
	}
	d, err := c.Descriptor(ctx)
	if err != nil {
		return err
	}
	m.Config = d

	for l := range core.Layers(ctx) {
		d, err := l.Descriptor(ctx)
		if err != nil {
			return err
		}
		m.Layers = append(m.Layers, d)
	}

	return nil
}
