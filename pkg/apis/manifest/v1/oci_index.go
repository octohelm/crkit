package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type OciIndex specv1.Index

var _ Manifest = OciIndex{}

func (OciIndex) Type() string {
	return specv1.MediaTypeImageIndex
}

func (i OciIndex) References() iter.Seq[Descriptor] {
	return func(yield func(Descriptor) bool) {
		for _, l := range i.Manifests {
			if !yield(l) {
				return
			}
		}
	}
}
