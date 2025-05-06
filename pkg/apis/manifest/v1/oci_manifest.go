package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const MediaTypeImageManifest = specv1.MediaTypeImageManifest

type OciManifest specv1.Manifest

var _ Manifest = OciManifest{}

func (OciManifest) Type() string {
	return MediaTypeImageManifest
}

func (m OciManifest) References() iter.Seq[Descriptor] {
	return func(yield func(Descriptor) bool) {
		if !yield(m.Config) {
			return
		}
		for _, l := range m.Layers {
			if !yield(l) {
				return
			}
		}
	}
}
