package internal

import (
	"maps"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func MergeDescriptors(descriptors ...ocispecv1.Descriptor) *ocispecv1.Descriptor {
	switch len(descriptors) {
	case 0:
		return &ocispecv1.Descriptor{
			MediaType: ocispecv1.MediaTypeDescriptor,
		}
	case 1:
		d := descriptors[0]
		return &d
	default:
		d := &ocispecv1.Descriptor{
			MediaType: ocispecv1.MediaTypeDescriptor,
		}

		for _, dd := range descriptors {
			if dd.MediaType != "" {
				d.MediaType = dd.MediaType
			}

			if dd.ArtifactType != "" {
				d.ArtifactType = dd.ArtifactType
			}

			if dd.Digest != "" {
				d.Digest = dd.Digest
			}

			if dd.Size > 0 {
				d.Size = dd.Size
			}

			if dd.Platform != nil {
				d.Platform = dd.Platform
			}

			if dd.Annotations != nil {
				if d.Annotations == nil {
					d.Annotations = make(map[string]string)
				}

				maps.Copy(d.Annotations, dd.Annotations)
			}
		}

		return d
	}
}
