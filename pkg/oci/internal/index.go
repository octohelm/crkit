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

type IndexCore interface {
	Manifests(ctx context.Context) iter.Seq2[oci.Manifest, error]
}

type Index struct {
	idx  *ocispecv1.Index
	desc *ocispecv1.Descriptor
	raw  []byte
}

func (i *Index) Descriptor(ctx context.Context) (ocispecv1.Descriptor, error) {
	return *i.desc, nil
}

func (i *Index) Raw(ctx context.Context) ([]byte, error) {
	return i.raw, nil
}

func (i *Index) Value(ctx context.Context) (ocispecv1.Index, error) {
	return *i.idx, nil
}

func (i *Index) InitFromReader(r io.Reader, descs ...ocispecv1.Descriptor) error {
	digester := digest.SHA256.Digester()
	buf := bytes.NewBuffer(nil)

	idx := &ocispecv1.Index{}

	if err := json.UnmarshalRead(io.TeeReader(r, io.MultiWriter(buf, digester.Hash())), idx); err != nil {
		return err
	}

	if len(descs) > 0 {
		i.desc = MergeDescriptors(descs...)
	} else {
		i.desc = &ocispecv1.Descriptor{
			MediaType:    idx.MediaType,
			ArtifactType: idx.ArtifactType,
		}
	}

	if d := digester.Digest(); d != i.desc.Digest {
		i.desc.Digest = d
	}

	i.desc.Size = int64(buf.Len())

	if len(idx.Annotations) > 0 {
		if i.desc.Annotations == nil {
			i.desc.Annotations = make(map[string]string)
		}
		maps.Copy(i.desc.Annotations, idx.Annotations)
	}

	i.idx = idx
	i.raw = buf.Bytes()

	return nil
}

func (i *Index) Build(mutates ...func(m *ocispecv1.Index) error) error {
	idx := &ocispecv1.Index{}
	i.idx = idx

	idx.SchemaVersion = 2
	idx.MediaType = ocispecv1.MediaTypeImageIndex

	for _, mut := range mutates {
		if err := mut(idx); err != nil {
			return err
		}
	}

	digester := digest.SHA256.Digester()
	buf := bytes.NewBuffer(nil)

	if err := json.MarshalWrite(
		io.MultiWriter(buf, digester.Hash()),
		idx,
		json.Deterministic(true),
	); err != nil {
		return err
	}

	d := &ocispecv1.Descriptor{
		MediaType:    idx.MediaType,
		ArtifactType: idx.ArtifactType,
		Digest:       digester.Digest(),
		Size:         int64(buf.Len()),
	}

	if len(idx.Annotations) > 0 {
		d.Annotations = make(map[string]string)
		maps.Copy(d.Annotations, idx.Annotations)
	}

	i.desc = d
	i.raw = buf.Bytes()

	return nil
}

func (i *Index) CollectTo(ctx context.Context, core IndexCore, m *ocispecv1.Index) error {
	for l := range core.Manifests(ctx) {
		d, err := l.Descriptor(ctx)
		if err != nil {
			return err
		}
		m.Manifests = append(m.Manifests, d)
	}

	return nil
}
