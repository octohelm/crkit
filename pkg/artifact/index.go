package artifact

import (
	"encoding/json"
	"sync/atomic"

	"github.com/google/go-containerregistry/pkg/v1/types"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func IndexWithArtifactType(base v1.ImageIndex, artifactType string, optFns ...Option) (v1.ImageIndex, error) {
	idx := &indexWithArtifactType{
		artifactType: artifactType,
		base:         base,
	}

	for _, optFn := range optFns {
		optFn(idx)
	}

	return idx, nil
}

type indexWithArtifactType struct {
	base         v1.ImageIndex
	artifactType string
	m            atomic.Pointer[specv1.Index]
	annotations  map[string]string
}

func (idx *indexWithArtifactType) SetAnnotation(k string, v string) {
	if idx.annotations == nil {
		idx.annotations = map[string]string{}
	}
	idx.annotations[k] = v
}

func (idx *indexWithArtifactType) MediaType() (types.MediaType, error) {
	return idx.base.MediaType()
}

func (idx *indexWithArtifactType) Digest() (v1.Hash, error) {
	return partial.Digest(idx)
}

func (idx *indexWithArtifactType) Size() (int64, error) {
	return partial.Size(idx)
}

func (idx *indexWithArtifactType) Image(hash v1.Hash) (v1.Image, error) {
	return idx.base.Image(hash)
}

func (idx *indexWithArtifactType) ImageIndex(hash v1.Hash) (v1.ImageIndex, error) {
	return idx.base.ImageIndex(hash)
}

func (idx *indexWithArtifactType) ArtifactType() (string, error) {
	return idx.artifactType, nil
}

func (idx *indexWithArtifactType) RawManifest() ([]byte, error) {
	m, err := idx.OCIIndex()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(m, "", "  ")
}

func (idx *indexWithArtifactType) IndexManifest() (*v1.IndexManifest, error) {
	i, err := idx.OCIIndex()
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	index := &v1.IndexManifest{}
	if err := json.Unmarshal(raw, index); err != nil {
		return nil, err
	}
	return index, nil
}

func (idx *indexWithArtifactType) OCIIndex() (*specv1.Index, error) {
	if m := idx.m.Load(); m != nil {
		return m, nil
	}

	raw, err := idx.base.RawManifest()
	if err != nil {
		return nil, err
	}

	i := &specv1.Index{}
	if err := json.Unmarshal(raw, i); err != nil {
		return nil, err
	}

	i.ArtifactType = idx.artifactType
	i.Annotations = idx.annotations

	idx.m.Store(i)
	return i, nil
}
