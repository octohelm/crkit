package partial

import (
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci/internal"
)

func MergeDescriptors(descriptors ...ocispecv1.Descriptor) ocispecv1.Descriptor {
	return *internal.MergeDescriptors(descriptors...)
}
