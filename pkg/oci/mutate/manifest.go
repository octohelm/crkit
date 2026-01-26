package mutate

import (
	"github.com/containerd/containerd/v2/core/images"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/crkit/pkg/oci"
)

func WithArtifactType[M oci.Manifest](base M, artifactType string) (M, error) {
	if artifactType != "" {
		switch x := any(base).(type) {
		case oci.Index:
			return any(&index{Index: x, artifactType: artifactType}).(M), nil
		case oci.Image:
			return any(&image{Image: x, artifactType: artifactType}).(M), nil
		}
	}
	return base, nil
}

func WithAnnotations[M oci.Manifest](base M, annotations map[string]string) (M, error) {
	if len(annotations) > 0 {
		switch x := any(base).(type) {
		case oci.Index:
			return any(&index{Index: x, annotations: annotations}).(M), nil
		case oci.Image:
			return any(&image{Image: x, annotations: annotations}).(M), nil
		}
	}
	return base, nil
}

func AnnotateOpenContainerImageName[M oci.Manifest](base M, name string, ref string) (M, error) {
	ann := map[string]string{}

	if ref != "" {
		ann[ocispecv1.AnnotationRefName] = ref
	}

	if name != "" {
		ann[ocispecv1.AnnotationBaseImageName] = name
	}

	return WithAnnotations(base, ann)
}

func AnnotateContainerdImageName[M oci.Manifest](base M, fullName string) (M, error) {
	if fullName != "" {
		return WithAnnotations(base, map[string]string{
			images.AnnotationImageName: fullName,
		})
	}
	return base, nil
}
