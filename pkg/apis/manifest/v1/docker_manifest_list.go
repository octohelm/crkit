package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerMediaTypeManifestList Docker 清单列表媒体类型
const DockerMediaTypeManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"

// DockerManifestList Docker 清单列表，用于多架构镜像分发
type DockerManifestList specv1.Index

var _ Manifest = DockerManifestList{}

func (DockerManifestList) Type() string {
	return DockerMediaTypeManifestList
}

func (i DockerManifestList) References() iter.Seq[Descriptor] {
	return func(yield func(Descriptor) bool) {
		for _, l := range i.Manifests {
			if !yield(l) {
				return
			}
		}
	}
}
