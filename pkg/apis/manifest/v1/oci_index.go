package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// MediaTypeImageIndex OCI 镜像索引媒体类型
const MediaTypeImageIndex = specv1.MediaTypeImageIndex

// OciIndex OCI 镜像索引，用于多架构镜像分发
type OciIndex specv1.Index

var _ Manifest = OciIndex{}

func (OciIndex) Type() string {
	return MediaTypeImageIndex
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
