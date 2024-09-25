package v1

import (
	"iter"

	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const DockerMediaTypeManifest = "application/vnd.docker.distribution.manifest.v2+json"

type DockerManifest specv1.Manifest

var _ Manifest = DockerManifest{}

func (DockerManifest) Type() string {
	return DockerMediaTypeManifest
}

func (m DockerManifest) References() iter.Seq[Descriptor] {
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
